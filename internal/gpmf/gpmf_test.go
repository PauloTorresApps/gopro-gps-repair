package gpmf

import "testing"

func TestParseOffsetsAndPutInt(t *testing.T) {
	// DEVC { STRM { GPSF L 4x1 = 2 ; GPS5 l 20x1 } }
	inner := []byte{
		'G', 'P', 'S', 'F', 'L', 4, 0, 1, 0, 0, 0, 2,
		'G', 'P', 'S', '5', 'l', 20, 0, 1,
		0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 4, 0, 0, 0, 5,
	}
	strm := append([]byte{'S', 'T', 'R', 'M', 0, 1, 0, byte(len(inner))}, inner...)
	buf := append([]byte{'D', 'E', 'V', 'C', 0, 1, 0, byte(len(strm))}, strm...)

	items, err := Parse(buf)
	if err != nil {
		t.Fatal(err)
	}
	s := Streams(items)
	if len(s) != 1 {
		t.Fatalf("esperava 1 STRM, veio %d", len(s))
	}
	f, _ := s[0].Find("GPSF")
	if v, _ := f.Ints(); v[0] != 2 {
		t.Fatalf("GPSF = %v", v)
	}
	// O offset aponta para o conteúdo no buffer original: regravar ali muda o valor.
	if err := PutInt('L', buf[f.Off:], 3); err != nil {
		t.Fatal(err)
	}
	items, _ = Parse(buf)
	f, _ = Streams(items)[0].Find("GPSF")
	if v, _ := f.Ints(); v[0] != 3 {
		t.Fatalf("após regravar, GPSF = %v", v)
	}
	g, _ := Streams(items)[0].Find("GPS5")
	if v, _ := g.Ints(); len(v) != 5 || v[4] != 5 {
		t.Fatalf("GPS5 = %v", v)
	}
	if err := PutInt('S', make([]byte, 2), 70000); err == nil {
		t.Fatal("esperava erro de estouro")
	}
}
