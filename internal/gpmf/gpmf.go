// Package gpmf decodifica o formato KLV da telemetria GoPro (GPMF),
// guardando a posição de cada valor para que ele possa ser regravado no lugar.
package gpmf

import (
	"encoding/binary"
	"fmt"
)

// KLV é um item GPMF: chave de 4 letras, tipo, tamanho da estrutura e
// repetições. Itens de tipo 0 são contêineres (DEVC, STRM...).
type KLV struct {
	Key      string
	Type     byte
	Size     int // bytes por estrutura
	Repeat   int
	Off      int // posição do conteúdo no buffer analisado
	Data     []byte
	Children []KLV
}

// Parse decodifica buf. base é somado aos offsets (útil para recursão).
func Parse(buf []byte) ([]KLV, error) { return parse(buf, 0) }

func parse(buf []byte, base int) ([]KLV, error) {
	var out []KLV
	for pos := 0; pos < len(buf); {
		if len(buf)-pos < 8 {
			// Alguns payloads terminam com bytes de preenchimento.
			break
		}
		key := string(buf[pos : pos+4])
		if buf[pos] == 0 && buf[pos+1] == 0 {
			break // preenchimento com zeros
		}
		k := KLV{
			Key:    key,
			Type:   buf[pos+4],
			Size:   int(buf[pos+5]),
			Repeat: int(binary.BigEndian.Uint16(buf[pos+6 : pos+8])),
			Off:    base + pos + 8,
		}
		n := k.Size * k.Repeat
		if pos+8+n > len(buf) {
			return nil, fmt.Errorf("GPMF truncado em %s (offset %d)", key, base+pos)
		}
		k.Data = buf[pos+8 : pos+8+n]
		if k.Type == 0 {
			kids, err := parse(k.Data, k.Off)
			if err != nil {
				return nil, err
			}
			k.Children = kids
		}
		out = append(out, k)
		pos += 8 + (n+3)&^3
	}
	return out, nil
}

// Find devolve o primeiro filho com a chave informada.
func (k KLV) Find(key string) (KLV, bool) {
	for _, c := range k.Children {
		if c.Key == key {
			return c, true
		}
	}
	return KLV{}, false
}

// Streams percorre a árvore e devolve todos os STRM.
func Streams(items []KLV) []KLV {
	var out []KLV
	for _, it := range items {
		if it.Key == "STRM" {
			out = append(out, it)
		}
		out = append(out, Streams(it.Children)...)
	}
	return out
}

// typeWidth é o tamanho em bytes dos tipos numéricos usados aqui.
func typeWidth(t byte) int {
	switch t {
	case 'b', 'B':
		return 1
	case 's', 'S':
		return 2
	case 'l', 'L', 'f':
		return 4
	case 'j', 'J', 'd':
		return 8
	}
	return 0
}

// Ints decodifica um item numérico inteiro em valores.
func (k KLV) Ints() ([]int64, error) {
	w := typeWidth(k.Type)
	if w == 0 || k.Type == 'f' || k.Type == 'd' {
		return nil, fmt.Errorf("%s: tipo %q não é inteiro", k.Key, k.Type)
	}
	var out []int64
	for i := 0; i+w <= len(k.Data); i += w {
		out = append(out, readInt(k.Type, k.Data[i:]))
	}
	return out, nil
}

func readInt(t byte, b []byte) int64 {
	switch t {
	case 'b':
		return int64(int8(b[0]))
	case 'B':
		return int64(b[0])
	case 's':
		return int64(int16(binary.BigEndian.Uint16(b)))
	case 'S':
		return int64(binary.BigEndian.Uint16(b))
	case 'l':
		return int64(int32(binary.BigEndian.Uint32(b)))
	case 'L':
		return int64(binary.BigEndian.Uint32(b))
	case 'j':
		return int64(binary.BigEndian.Uint64(b))
	case 'J':
		return int64(binary.BigEndian.Uint64(b))
	}
	return 0
}

// PutInt grava v em b no formato do tipo t. Falha se não couber.
func PutInt(t byte, b []byte, v int64) error {
	fits := func(lo, hi int64) error {
		if v < lo || v > hi {
			return fmt.Errorf("valor %d não cabe no tipo %q", v, t)
		}
		return nil
	}
	switch t {
	case 'b':
		if err := fits(-128, 127); err != nil {
			return err
		}
		b[0] = byte(int8(v))
	case 'B':
		if err := fits(0, 255); err != nil {
			return err
		}
		b[0] = byte(v)
	case 's':
		if err := fits(-32768, 32767); err != nil {
			return err
		}
		binary.BigEndian.PutUint16(b, uint16(int16(v)))
	case 'S':
		if err := fits(0, 65535); err != nil {
			return err
		}
		binary.BigEndian.PutUint16(b, uint16(v))
	case 'l':
		if err := fits(-1<<31, 1<<31-1); err != nil {
			return err
		}
		binary.BigEndian.PutUint32(b, uint32(int32(v)))
	case 'L':
		if err := fits(0, 1<<32-1); err != nil {
			return err
		}
		binary.BigEndian.PutUint32(b, uint32(v))
	default:
		return fmt.Errorf("tipo %q não suportado para escrita", t)
	}
	return nil
}
