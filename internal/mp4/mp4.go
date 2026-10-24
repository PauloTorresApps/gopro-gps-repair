// Package mp4 localiza, dentro de um MP4 da GoPro, as amostras da trilha de
// telemetria (gpmd): onde cada uma está no arquivo, quantos bytes tem e em que
// instante do vídeo começa. Só lê o índice (moov); o conteúdo das amostras é
// lido por quem chama.
package mp4

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Sample é uma amostra da trilha gpmd (um "payload" GPMF, ~1 s de telemetria).
type Sample struct {
	Offset int64   // posição absoluta no arquivo
	Size   int64   // bytes
	Time   float64 // s desde o início da trilha
	Dur    float64 // s
}

type box struct {
	typ   string
	start int64 // início do cabeçalho
	data  int64 // início do conteúdo
	end   int64 // fim do box
}

// readBoxes lista os boxes entre start e end.
func readBoxes(r io.ReaderAt, start, end int64) ([]box, error) {
	var out []box
	hdr := make([]byte, 16)
	for pos := start; pos+8 <= end; {
		if _, err := r.ReadAt(hdr[:8], pos); err != nil {
			return nil, fmt.Errorf("lendo box em %d: %w", pos, err)
		}
		size := int64(binary.BigEndian.Uint32(hdr[0:4]))
		typ := string(hdr[4:8])
		data := pos + 8
		switch size {
		case 1: // tamanho de 64 bits logo após o tipo
			if _, err := r.ReadAt(hdr[8:16], pos+8); err != nil {
				return nil, fmt.Errorf("lendo box %s: %w", typ, err)
			}
			size = int64(binary.BigEndian.Uint64(hdr[8:16]))
			data = pos + 16
		case 0: // vai até o fim do contêiner
			size = end - pos
		}
		if size < data-pos || pos+size > end {
			return nil, fmt.Errorf("box %s com tamanho inválido (%d) em %d", typ, size, pos)
		}
		out = append(out, box{typ: typ, start: pos, data: data, end: pos + size})
		pos += size
	}
	return out, nil
}

func child(boxes []box, typ string) (box, bool) {
	for _, b := range boxes {
		if b.typ == typ {
			return b, true
		}
	}
	return box{}, false
}

func readPayload(r io.ReaderAt, b box) ([]byte, error) {
	buf := make([]byte, b.end-b.data)
	if _, err := r.ReadAt(buf, b.data); err != nil {
		return nil, fmt.Errorf("lendo %s: %w", b.typ, err)
	}
	return buf, nil
}

// GPMDSamples abre o MP4 e devolve as amostras da trilha de telemetria GoPro.
func GPMDSamples(path string) ([]Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}

	top, err := readBoxes(f, 0, st.Size())
	if err != nil {
		return nil, err
	}
	moov, ok := child(top, "moov")
	if !ok {
		return nil, fmt.Errorf("arquivo sem box moov (não é um MP4 válido?)")
	}
	traks, err := readBoxes(f, moov.data, moov.end)
	if err != nil {
		return nil, err
	}

	for _, trak := range traks {
		if trak.typ != "trak" {
			continue
		}
		samples, isGPMD, err := parseTrak(f, trak)
		if err != nil {
			return nil, err
		}
		if isGPMD {
			return samples, nil
		}
	}
	return nil, fmt.Errorf("vídeo sem trilha de telemetria GoPro (gpmd)")
}

// parseTrak devolve as amostras se a trilha for gpmd.
func parseTrak(f io.ReaderAt, trak box) ([]Sample, bool, error) {
	path := func(parent box, names ...string) (box, error) {
		cur := parent
		for _, n := range names {
			kids, err := readBoxes(f, cur.data, cur.end)
			if err != nil {
				return box{}, err
			}
			b, ok := child(kids, n)
			if !ok {
				return box{}, errMissing
			}
			cur = b
		}
		return cur, nil
	}

	stsd, err := path(trak, "mdia", "minf", "stbl", "stsd")
	if err == errMissing {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	sd, err := readPayload(f, stsd)
	if err != nil {
		return nil, false, err
	}
	// versão/flags (4), número de entradas (4), 1ª entrada: tamanho (4), formato (4)
	if len(sd) < 16 || string(sd[12:16]) != "gpmd" {
		return nil, false, nil
	}

	mdhd, err := path(trak, "mdia", "mdhd")
	if err != nil {
		return nil, true, fmt.Errorf("trilha gpmd sem mdhd: %w", err)
	}
	md, err := readPayload(f, mdhd)
	if err != nil {
		return nil, true, err
	}
	var timescale uint32
	if len(md) > 0 && md[0] == 1 {
		timescale = be32(md, 20)
	} else {
		timescale = be32(md, 12)
	}
	if timescale == 0 {
		return nil, true, fmt.Errorf("trilha gpmd com timescale zero")
	}

	stbl, err := path(trak, "mdia", "minf", "stbl")
	if err != nil {
		return nil, true, err
	}
	kids, err := readBoxes(f, stbl.data, stbl.end)
	if err != nil {
		return nil, true, err
	}
	load := func(n string) ([]byte, bool, error) {
		b, ok := child(kids, n)
		if !ok {
			return nil, false, nil
		}
		p, err := readPayload(f, b)
		return p, true, err
	}

	stsz, ok, err := load("stsz")
	if err != nil || !ok {
		return nil, true, fmt.Errorf("trilha gpmd sem stsz: %v", err)
	}
	stsc, ok, err := load("stsc")
	if err != nil || !ok {
		return nil, true, fmt.Errorf("trilha gpmd sem stsc: %v", err)
	}
	stts, ok, err := load("stts")
	if err != nil || !ok {
		return nil, true, fmt.Errorf("trilha gpmd sem stts: %v", err)
	}
	var chunks []int64
	if co, ok, err := load("stco"); err != nil {
		return nil, true, err
	} else if ok {
		n := int(be32(co, 4))
		for i := 0; i < n; i++ {
			chunks = append(chunks, int64(be32(co, 8+4*i)))
		}
	} else if co, ok, err := load("co64"); err != nil {
		return nil, true, err
	} else if ok {
		n := int(be32(co, 4))
		for i := 0; i < n; i++ {
			chunks = append(chunks, int64(binary.BigEndian.Uint64(co[8+8*i:])))
		}
	} else {
		return nil, true, fmt.Errorf("trilha gpmd sem stco/co64")
	}

	samples, err := buildSamples(stsz, stsc, stts, chunks, float64(timescale))
	return samples, true, err
}

var errMissing = fmt.Errorf("box ausente")

// buildSamples cruza as tabelas de tamanho (stsz), de amostras por chunk
// (stsc), de duração (stts) e de posição dos chunks (stco/co64).
func buildSamples(stsz, stsc, stts []byte, chunks []int64, timescale float64) ([]Sample, error) {
	fixed := int64(be32(stsz, 4))
	count := int(be32(stsz, 8))
	sizes := make([]int64, count)
	for i := range sizes {
		if fixed != 0 {
			sizes[i] = fixed
		} else {
			sizes[i] = int64(be32(stsz, 12+4*i))
		}
	}

	type run struct{ first, perChunk int }
	var runs []run
	for i, n := 0, int(be32(stsc, 4)); i < n; i++ {
		runs = append(runs, run{int(be32(stsc, 8+12*i)), int(be32(stsc, 12+12*i))})
	}

	samples := make([]Sample, 0, count)
	s := 0
	for c := range chunks {
		per := 0
		for _, r := range runs { // a última faixa que começa até este chunk vale
			if r.first <= c+1 {
				per = r.perChunk
			}
		}
		off := chunks[c]
		for k := 0; k < per && s < count; k++ {
			samples = append(samples, Sample{Offset: off, Size: sizes[s]})
			off += sizes[s]
			s++
		}
	}
	if s != count {
		return nil, fmt.Errorf("índice gpmd inconsistente: %d de %d amostras localizadas", s, count)
	}

	var t int64
	i := 0
	for e, n := 0, int(be32(stts, 4)); e < n; e++ {
		cnt := int(be32(stts, 8+8*e))
		delta := int64(be32(stts, 12+8*e))
		for k := 0; k < cnt && i < len(samples); k++ {
			samples[i].Time = float64(t) / timescale
			samples[i].Dur = float64(delta) / timescale
			t += delta
			i++
		}
	}
	return samples, nil
}

func be32(b []byte, off int) uint32 {
	if off+4 > len(b) {
		return 0
	}
	return binary.BigEndian.Uint32(b[off:])
}

// TimeField é o par criação+modificação de um box mvhd, tkhd ou mdhd: os dois
// valores são contíguos e têm tamanho fixo (8 bytes na versão 0, 16 na 1).
type TimeField struct {
	Off  int64
	Len  int
	Wide bool // versão 1: valores de 64 bits
}

// TimeFields localiza as datas de criação/modificação do contêiner (mvhd e,
// de cada trilha, tkhd e mdhd). Não inclui nenhum metadado da GoPro.
func TimeFields(path string) ([]TimeField, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	top, err := readBoxes(f, 0, st.Size())
	if err != nil {
		return nil, err
	}
	moov, ok := child(top, "moov")
	if !ok {
		return nil, fmt.Errorf("arquivo sem box moov (não é um MP4 válido?)")
	}

	var out []TimeField
	add := func(b box) error {
		var v [1]byte
		if _, err := f.ReadAt(v[:], b.data); err != nil {
			return fmt.Errorf("lendo %s: %w", b.typ, err)
		}
		tf := TimeField{Off: b.data + 4, Len: 8}
		if v[0] == 1 {
			tf.Len, tf.Wide = 16, true
		}
		if tf.Off+int64(tf.Len) > b.end {
			return fmt.Errorf("box %s curto demais", b.typ)
		}
		out = append(out, tf)
		return nil
	}

	kids, err := readBoxes(f, moov.data, moov.end)
	if err != nil {
		return nil, err
	}
	for _, k := range kids {
		switch k.typ {
		case "mvhd":
			if err := add(k); err != nil {
				return nil, err
			}
		case "trak":
			tk, err := readBoxes(f, k.data, k.end)
			if err != nil {
				return nil, err
			}
			if b, ok := child(tk, "tkhd"); ok {
				if err := add(b); err != nil {
					return nil, err
				}
			}
			if mdia, ok := child(tk, "mdia"); ok {
				mk, err := readBoxes(f, mdia.data, mdia.end)
				if err != nil {
					return nil, err
				}
				if b, ok := child(mk, "mdhd"); ok {
					if err := add(b); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return out, nil
}
