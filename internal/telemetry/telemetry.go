// Package telemetry lê os blocos de GPS (GPS5) de um MP4 GoPro e guarda, para
// cada campo, a posição exata no arquivo, permitindo regravá-lo no lugar.
package telemetry

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"gopro-gps-repair/internal/gpmf"
	"gopro-gps-repair/internal/mp4"
)

// Field aponta para o conteúdo de um item GPMF dentro do arquivo.
type Field struct {
	Off  int64 // posição absoluta no arquivo
	Type byte
	Len  int // bytes
}

// Block é o GPS de um payload (~1 s): cabeçalho e amostras.
type Block struct {
	Index       int
	PayloadTime float64 // s desde o início do vídeo
	PayloadDur  float64

	GPSURaw   string
	GPSU      time.Time
	GPSUValid bool
	GPSUField *Field
	// Time é o horário efetivo do bloco. Igual ao GPSU, exceto quando ele falta
	// ou destoa do relógio do vídeo (TimeSuspect).
	Time        time.Time
	TimeSuspect bool

	Fix      int // GPSF: 0 sem fix, 2 = 2D, 3 = 3D; -1 ausente
	FixField *Field
	DOP      float64 // GPSP/100; NaN se ausente
	DOPField *Field

	Scale [5]float64
	GPS5  Field
	First int // índice da 1ª amostra em Track.Samples
	Count int
}

// Sample é uma amostra GPS5, já em unidades físicas.
type Sample struct {
	Block   int
	Time    time.Time
	Lat     float64 // graus
	Lon     float64
	Alt     float64 // m
	Speed2D float64 // m/s
	Speed3D float64
	Valid   bool // coordenadas dentro da faixa e diferentes de (0, 0)
}

// Track é toda a telemetria GPS de um arquivo.
type Track struct {
	Path    string
	Blocks  []Block
	Samples []Sample
}

const gpsuLayout = "060102150405.000"

// Load lê o GPS do MP4.
func Load(path string) (*Track, error) {
	payloads, err := mp4.GPMDSamples(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tr := &Track{Path: path}
	scale := [5]float64{1, 1, 1, 1, 1}
	fix := -1
	for _, p := range payloads {
		buf := make([]byte, p.Size)
		if _, err := f.ReadAt(buf, p.Offset); err != nil {
			return nil, fmt.Errorf("lendo payload em %d: %w", p.Offset, err)
		}
		items, err := gpmf.Parse(buf)
		if err != nil {
			return nil, err
		}
		for _, strm := range gpmf.Streams(items) {
			gps5, ok := strm.Find("GPS5")
			if !ok {
				continue
			}
			field := func(k gpmf.KLV) *Field {
				return &Field{Off: p.Offset + int64(k.Off), Type: k.Type, Len: len(k.Data)}
			}
			b := Block{
				Index:       len(tr.Blocks),
				PayloadTime: p.Time,
				PayloadDur:  p.Dur,
				DOP:         math.NaN(),
				GPS5:        *field(gps5),
				First:       len(tr.Samples),
			}

			// GPSF e SCAL são "pegajosos": se um payload não os repete, vale o
			// último visto.
			if k, ok := strm.Find("GPSF"); ok {
				if v, err := k.Ints(); err == nil && len(v) > 0 {
					fix = int(v[0])
					b.FixField = field(k)
				}
			}
			b.Fix = fix
			if k, ok := strm.Find("GPSP"); ok {
				if v, err := k.Ints(); err == nil && len(v) > 0 {
					b.DOP = float64(v[0]) / 100
					b.DOPField = field(k)
				}
			}
			if k, ok := strm.Find("SCAL"); ok {
				if v, err := k.Ints(); err == nil {
					for i := 0; i < len(v) && i < 5; i++ {
						if v[i] != 0 {
							scale[i] = float64(v[i])
						}
					}
				}
			}
			b.Scale = scale
			if k, ok := strm.Find("GPSU"); ok {
				b.GPSURaw = strings.TrimSpace(strings.TrimRight(string(k.Data), "\x00"))
				b.GPSUField = field(k)
				if t, err := time.Parse(gpsuLayout, b.GPSURaw); err == nil {
					b.GPSU, b.GPSUValid = t.UTC(), true
				}
			}

			if gps5.Type != 'l' {
				return nil, fmt.Errorf("GPS5 com tipo %q não suportado", gps5.Type)
			}
			n := len(gps5.Data) / 20
			for i := 0; i < n; i++ {
				v, _ := gpmf.KLV{Type: 'l', Data: gps5.Data[i*20 : i*20+20]}.Ints()
				s := Sample{
					Block:   b.Index,
					Lat:     float64(v[0]) / scale[0],
					Lon:     float64(v[1]) / scale[1],
					Alt:     float64(v[2]) / scale[2],
					Speed2D: float64(v[3]) / scale[3],
					Speed3D: float64(v[4]) / scale[4],
				}
				s.Valid = !(s.Lat == 0 && s.Lon == 0) && math.Abs(s.Lat) <= 90 && math.Abs(s.Lon) <= 180
				tr.Samples = append(tr.Samples, s)
			}
			b.Count = n
			tr.Blocks = append(tr.Blocks, b)
		}
	}
	if len(tr.Blocks) == 0 {
		return nil, fmt.Errorf("telemetria sem blocos GPS5 (câmera sem GPS ou GPS desligado)")
	}
	if err := tr.assignTimes(); err != nil {
		return nil, err
	}
	return tr, nil
}

// assignTimes define o horário de cada bloco e de cada amostra. O relógio do
// vídeo (instante do payload) é regular; o GPSU dá o horário absoluto. A
// diferença mediana entre os dois, nos blocos com fix, ancora o relógio do
// vídeo, e um GPSU que destoe mais de suspectGap dessa âncora é descartado.
//
// Dentro do bloco, as amostras são distribuídas até o horário do bloco
// seguinte, como faz o strava-overlay ao ler o arquivo.
func (tr *Track) assignTimes() error {
	const suspectGap = 0.5 // s

	var diffs []float64
	for _, withFix := range []bool{true, false} {
		for _, b := range tr.Blocks {
			if b.GPSUValid && (!withFix || b.Fix >= 2) {
				diffs = append(diffs, float64(b.GPSU.UnixNano())/1e9-b.PayloadTime)
			}
		}
		if len(diffs) > 0 {
			break
		}
	}
	if len(diffs) == 0 {
		return fmt.Errorf("nenhum bloco com horário GPS (GPSU) válido")
	}
	sort.Float64s(diffs)
	anchor := diffs[len(diffs)/2]

	for i := range tr.Blocks {
		b := &tr.Blocks[i]
		exp := anchor + b.PayloadTime
		expected := time.Unix(0, int64(math.Round(exp*1e9))).UTC().Truncate(time.Millisecond)
		if b.GPSUValid && math.Abs(float64(b.GPSU.UnixNano())/1e9-exp) <= suspectGap {
			b.Time = b.GPSU
		} else {
			b.Time = expected
			b.TimeSuspect = true
		}
	}

	for i := range tr.Blocks {
		b := &tr.Blocks[i]
		if b.Count == 0 {
			continue
		}
		span := time.Duration(b.PayloadDur * float64(time.Second))
		if i+1 < len(tr.Blocks) {
			if d := tr.Blocks[i+1].Time.Sub(b.Time); d > 0 {
				span = d
			}
		}
		step := span / time.Duration(b.Count)
		for j := 0; j < b.Count; j++ {
			tr.Samples[b.First+j].Time = b.Time.Add(time.Duration(j) * step)
		}
	}
	return nil
}
