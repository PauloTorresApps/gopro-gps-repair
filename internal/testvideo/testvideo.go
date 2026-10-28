// Package testvideo gera um MP4 sintético no formato da GoPro, com uma trilha
// de telemetria (gpmd) cujo GPS tem os defeitos típicos de trilhas sob árvores:
// início sem fix, fix 2D, DOP alto, saltos de posição e um horário GPSU
// inconsistente. Serve aos testes e para experimentar a interface sem um vídeo
// real.
package testvideo

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"time"
)

// Options controla o vídeo gerado.
type Options struct {
	Seconds int   // duração da trilha (padrão 60)
	Filler  int64 // bytes de "vídeo" (zeros) antes da telemetria, para simular arquivos grandes
}

const (
	perBlock  = 18
	timescale = 1000
)

// Write grava o vídeo em path.
func Write(path string, opt Options) error {
	if opt.Seconds <= 0 {
		opt.Seconds = 60
	}
	var payloads [][]byte
	start := time.Date(2025, 8, 16, 20, 18, 9, 0, time.UTC)
	for i := 0; i < opt.Seconds; i++ {
		payloads = append(payloads, block(i, opt.Seconds, start))
	}

	ftyp := box("ftyp", []byte("mp41\x00\x00\x00\x00mp41isom"))
	var mdatBody int64 = opt.Filler
	for _, p := range payloads {
		mdatBody += int64(len(p))
	}
	mdatHdr := make([]byte, 16) // tamanho de 64 bits, para aceitar arquivos grandes
	binary.BigEndian.PutUint32(mdatHdr[0:], 1)
	copy(mdatHdr[4:], "mdat")
	binary.BigEndian.PutUint64(mdatHdr[8:], uint64(16+mdatBody))

	first := int64(len(ftyp)) + 16 + opt.Filler
	moov := moovBox(payloads, first, opt.Seconds)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := func(b []byte) {
		if err == nil {
			_, err = f.Write(b)
		}
	}
	w(ftyp)
	w(mdatHdr)
	if opt.Filler > 0 && err == nil {
		// Pula os zeros (arquivo esparso): rápido e sem ocupar disco.
		_, err = f.Seek(opt.Filler, 1)
	}
	for _, p := range payloads {
		w(p)
	}
	w(moov)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// block monta o payload GPMF do segundo i.
func block(i, n int, start time.Time) []byte {
	fix, dop := 3, 1.5
	noise, jump := 0.0, 0.0
	switch {
	case i < 3:
		fix, dop = 0, 99.99
	case i < 5:
		fix, dop, noise = 2, 8, 6
	case i >= n/3 && i < n/3+6: // sob as árvores
		dop, noise = 12, 15
	case i == 2*n/3:
		jump = 40
	}
	gpsu := start.Add(time.Duration(i) * time.Second)
	if i == n/2 {
		gpsu = gpsu.Add(-37 * time.Second) // horário inconsistente
	}

	scale := []int32{10000000, 10000000, 1000, 1000, 100}
	var gps5 bytes.Buffer
	for j := 0; j < perBlock; j++ {
		t := float64(i) + float64(j)/perBlock
		lat, lon, alt, v := route(t)
		if fix == 0 {
			lat, lon, alt = 0, 0, 0
		} else {
			// Ruído determinístico, proporcional ao DOP do trecho.
			lat += noise * math.Sin(t*7.3) / 111320
			lon += (noise*math.Cos(t*5.1) + jump) / (111320 * math.Cos(lat*math.Pi/180))
			alt += 3 * noise * math.Sin(t*3.7)
		}
		for c, x := range []float64{lat, lon, alt, v, v} {
			binary.Write(&gps5, binary.BigEndian, int32(math.Round(x*float64(scale[c]))))
		}
	}
	var sc bytes.Buffer
	for _, s := range scale {
		binary.Write(&sc, binary.BigEndian, s)
	}

	strm := klv("STNM", 'c', 1, []byte("GPS (Lat., Long., Alt., 2D speed, 3D speed)"))
	strm = append(strm, klv("GPSF", 'L', 4, u32(uint32(fix)))...)
	strm = append(strm, klv("GPSU", 'U', 16, []byte(gpsu.Format("060102150405.000")))...)
	strm = append(strm, klv("GPSP", 'S', 2, u16(uint16(math.Round(dop*100))))...)
	strm = append(strm, klv("SCAL", 'l', 4, sc.Bytes())...)
	strm = append(strm, klv("GPS5", 'l', 20, gps5.Bytes())...)

	devc := klv("DVID", 'L', 4, u32(1))
	devc = append(devc, klv("DVNM", 'c', 1, []byte("Camera"))...)
	devc = append(devc, klv("STRM", 0, 1, strm)...)
	return klv("DEVC", 0, 1, devc)
}

// route é o percurso real: uma curva suave a ~5 m/s, com uma subida leve.
func route(t float64) (lat, lon, alt, speed float64) {
	const lat0, lon0 = -10.1699, -48.2795
	x := 5 * t               // m para leste
	y := 30 * math.Sin(t/15) // m para norte
	lat = lat0 + y/111320
	lon = lon0 + x/(111320*math.Cos(lat0*math.Pi/180))
	alt = 360 + 0.1*t
	vy := 2 * math.Cos(t/15)
	return lat, lon, alt, math.Hypot(5, vy)
}

// klv monta um item GPMF; size é o tamanho de uma estrutura (1 para
// contêineres e textos).
func klv(key string, typ byte, size int, data []byte) []byte {
	out := make([]byte, 8, 8+len(data)+3)
	copy(out, key)
	out[4] = typ
	out[5] = byte(size)
	binary.BigEndian.PutUint16(out[6:], uint16(len(data)/size))
	out = append(out, data...)
	for len(out)%4 != 0 {
		out = append(out, 0)
	}
	return out
}

func moovBox(payloads [][]byte, first int64, seconds int) []byte {
	dur := uint32(seconds * timescale)

	mvhd := make([]byte, 100)
	binary.BigEndian.PutUint32(mvhd[12:], timescale)
	binary.BigEndian.PutUint32(mvhd[16:], dur)

	tkhd := make([]byte, 84)
	binary.BigEndian.PutUint32(tkhd[0:], 3) // habilitada
	binary.BigEndian.PutUint32(tkhd[12:], 1)

	mdhd := make([]byte, 24)
	binary.BigEndian.PutUint32(mdhd[12:], timescale)
	binary.BigEndian.PutUint32(mdhd[16:], dur)

	hdlr := append(make([]byte, 8), []byte("meta\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00GoPro MET\x00")...)

	stsd := u32(0, 1)
	stsd = append(stsd, box("gpmd", make([]byte, 8))...)
	stts := u32(0, 1, uint32(len(payloads)), timescale)
	stsc := u32(0, 1, 1, uint32(len(payloads)), 1) // um chunk com todas as amostras
	stsz := u32(0, 0, uint32(len(payloads)))
	for _, p := range payloads {
		stsz = append(stsz, u32(uint32(len(p)))...)
	}
	// co64: o primeiro payload pode estar além de 4 GiB.
	co64 := u32(0, 1)
	co64 = binary.BigEndian.AppendUint64(co64, uint64(first))

	stbl := concat(box("stsd", stsd), box("stts", stts), box("stsc", stsc), box("stsz", stsz), box("co64", co64))
	minf := box("minf", box("stbl", stbl))
	mdia := box("mdia", concat(box("mdhd", mdhd), box("hdlr", hdlr), minf))
	trak := box("trak", concat(box("tkhd", tkhd), mdia))
	return box("moov", concat(box("mvhd", mvhd), trak))
}

func box(typ string, body []byte) []byte {
	out := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(out, uint32(8+len(body)))
	copy(out[4:], typ)
	return append(out, body...)
}

func concat(parts ...[]byte) []byte { return bytes.Join(parts, nil) }

func u32(v ...uint32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.BigEndian.PutUint32(b[4*i:], x)
	}
	return b
}

func u16(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }
