// Package diagnose avalia a saúde do GPS gravado pela GoPro: fix, precisão
// (DOP), saltos de posição e de altitude, e horário dos blocos.
package diagnose

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gopro-gps-repair/internal/geo"
	"gopro-gps-repair/internal/telemetry"
)

// Limites usados na avaliação.
const (
	GoodDOP     = 2.0 // até aqui, precisão boa
	FairDOP     = 5.0 // acima disso, precisão ruim
	PoorDOP     = 10.0
	JumpExcessM = 5.0 // deslocamento além do que a velocidade explica
	AltJumpM    = 5.0 // variação de altitude entre amostras consecutivas
)

// BlockReport resume um bloco (~1 s).
type BlockReport struct {
	Index       int       `json:"indice"`
	VideoTime   float64   `json:"tempo_video_s"`
	Time        time.Time `json:"horario"`
	Fix         int       `json:"fix"`
	DOP         float64   `json:"dop"` // -1: ausente
	Samples     int       `json:"amostras"`
	Jumps       int       `json:"saltos_posicao"`
	AltJumps    int       `json:"saltos_altitude"`
	TimeSuspect bool      `json:"horario_suspeito"`
	Quality     float64   `json:"qualidade"` // 0 a 1
}

// Report é o diagnóstico completo de um arquivo.
type Report struct {
	Path        string        `json:"arquivo"`
	Start       time.Time     `json:"inicio"`
	End         time.Time     `json:"fim"`
	Duration    float64       `json:"duracao_s"`
	Blocks      int           `json:"blocos"`
	Samples     int           `json:"amostras"`
	NoFix       int           `json:"amostras_sem_fix"`
	Fix2D       int           `json:"amostras_fix_2d"`
	Fix3D       int           `json:"amostras_fix_3d"`
	HighDOP     int           `json:"amostras_dop_alto"`
	MedianDOP   float64       `json:"dop_mediano"`
	Jumps       int           `json:"saltos_posicao"`
	MaxJump     float64       `json:"maior_salto_m"`
	AltJumps    int           `json:"saltos_altitude"`
	MaxAltJump  float64       `json:"maior_salto_altitude_m"`
	TimeSuspect int           `json:"blocos_horario_suspeito"`
	DistanceM   float64       `json:"distancia_m"`
	Score       int           `json:"nota"`
	Verdict     string        `json:"veredito"`
	Problems    []string      `json:"problemas"`
	BlockList   []BlockReport `json:"lista_blocos"`
}

// Run avalia a trilha.
func Run(tr *telemetry.Track) *Report {
	r := &Report{Path: tr.Path, Blocks: len(tr.Blocks), Samples: len(tr.Samples)}
	s := tr.Samples
	if len(s) > 0 {
		r.Start, r.End = s[0].Time, s[len(s)-1].Time
		r.Duration = r.End.Sub(r.Start).Seconds()
	}

	usable := func(i int) bool { // posição aproveitável
		b := tr.Blocks[s[i].Block]
		return s[i].Valid && b.Fix >= 2
	}

	jump := make([]bool, len(s))
	altJump := make([]bool, len(s))
	for i := 1; i < len(s); i++ {
		if !usable(i) || !usable(i-1) {
			continue
		}
		dt := s[i].Time.Sub(s[i-1].Time).Seconds()
		d := geo.Distance(s[i-1].Lat, s[i-1].Lon, s[i].Lat, s[i].Lon)
		v := (s[i].Speed2D + s[i-1].Speed2D) / 2
		if excess := d - v*math.Max(dt, 0); excess > JumpExcessM {
			jump[i] = true
			r.Jumps++
		}
		r.MaxJump = math.Max(r.MaxJump, d)
		if dh := math.Abs(s[i].Alt - s[i-1].Alt); dh > AltJumpM && dt < 0.5 {
			altJump[i] = true
			r.AltJumps++
			r.MaxAltJump = math.Max(r.MaxAltJump, dh)
		}
		if tr.Blocks[s[i].Block].Fix >= 3 && !jump[i] {
			r.DistanceM += d
		}
	}

	var dops []float64
	var total float64
	for _, b := range tr.Blocks {
		br := BlockReport{
			Index: b.Index, VideoTime: b.PayloadTime, Time: b.Time, Fix: b.Fix,
			DOP: b.DOP, Samples: b.Count, TimeSuspect: b.TimeSuspect,
		}
		if math.IsNaN(br.DOP) {
			br.DOP = -1
		}
		if b.TimeSuspect {
			r.TimeSuspect++
		}
		var q float64
		for i := b.First; i < b.First+b.Count; i++ {
			switch {
			case !s[i].Valid || b.Fix < 2:
				r.NoFix++
			case b.Fix == 2:
				r.Fix2D++
			default:
				r.Fix3D++
			}
			if b.Fix >= 2 && b.DOP > FairDOP {
				r.HighDOP++
			}
			if b.Fix >= 2 && !math.IsNaN(b.DOP) {
				dops = append(dops, b.DOP)
			}
			if jump[i] {
				br.Jumps++
			}
			if altJump[i] {
				br.AltJumps++
			}
			sq := sampleQuality(s[i].Valid, b.Fix, b.DOP)
			if jump[i] {
				sq = 0
			} else if altJump[i] {
				sq *= 0.5
			}
			q += sq
		}
		total += q
		if b.Count > 0 {
			br.Quality = q / float64(b.Count)
		}
		r.BlockList = append(r.BlockList, br)
	}
	if len(dops) > 0 {
		sort.Float64s(dops)
		r.MedianDOP = dops[len(dops)/2]
	}
	if len(s) > 0 {
		r.Score = int(math.Round(100 * total / float64(len(s))))
	}
	switch {
	case r.Score >= 85:
		r.Verdict = "bom"
	case r.Score >= 50:
		r.Verdict = "degradado"
	default:
		r.Verdict = "ruim"
	}
	r.Problems = problems(r)
	return r
}

// sampleQuality dá o peso de uma amostra na nota: 1 para fix 3D com DOP bom,
// caindo com o DOP; 2D vale pouco (altitude não confiável); sem fix vale 0.
func sampleQuality(valid bool, fix int, dop float64) float64 {
	switch {
	case !valid || fix < 2:
		return 0
	case fix == 2:
		return 0.3
	case math.IsNaN(dop) || dop <= GoodDOP:
		return 1
	case dop <= FairDOP:
		return 0.8
	case dop <= PoorDOP:
		return 0.5
	}
	return 0.2
}

// problems descreve, em texto, os trechos com defeito.
func problems(r *Report) []string {
	var out []string
	add := func(desc string, pred func(BlockReport) bool) {
		ranges := segments(r.BlockList, pred)
		if len(ranges) > 0 {
			out = append(out, fmt.Sprintf("%s: %s", desc, strings.Join(ranges, ", ")))
		}
	}
	add("sem fix", func(b BlockReport) bool { return b.Fix < 2 })
	add("só fix 2D (altitude não confiável)", func(b BlockReport) bool { return b.Fix == 2 })
	add(fmt.Sprintf("precisão ruim (DOP > %.0f)", FairDOP), func(b BlockReport) bool { return b.Fix >= 2 && b.DOP > FairDOP })
	add("saltos de posição", func(b BlockReport) bool { return b.Jumps > 0 })
	add("saltos de altitude", func(b BlockReport) bool { return b.AltJumps > 0 })
	add("horário GPS (GPSU) inconsistente", func(b BlockReport) bool { return b.TimeSuspect })
	return out
}

// segments agrupa blocos consecutivos que satisfazem pred em faixas de tempo
// do vídeo, como "0,0–2,0 s".
func segments(blocks []BlockReport, pred func(BlockReport) bool) []string {
	var out []string
	for i := 0; i < len(blocks); {
		if !pred(blocks[i]) {
			i++
			continue
		}
		j := i
		for j+1 < len(blocks) && pred(blocks[j+1]) {
			j++
		}
		end := blocks[j].VideoTime + 1.0
		if j+1 < len(blocks) {
			end = blocks[j+1].VideoTime
		}
		out = append(out, strings.ReplaceAll(fmt.Sprintf("%.1f–%.1f s", blocks[i].VideoTime, end), ".", ","))
		i = j + 1
	}
	return out
}
