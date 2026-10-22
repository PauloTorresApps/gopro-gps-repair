package mp4

import (
	"encoding/binary"
	"testing"
)

func u32(v ...uint32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.BigEndian.PutUint32(b[4*i:], x)
	}
	return b
}

// Três amostras em dois chunks (2 + 1), tamanhos variáveis, 1 s cada.
func TestBuildSamples(t *testing.T) {
	stsz := u32(0, 0, 3, 100, 200, 300)
	stsc := u32(0, 2, 1, 2, 1, 2, 1, 1)
	stts := u32(0, 1, 3, 1000)
	got, err := buildSamples(stsz, stsc, stts, []int64{5000, 9000}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	want := []Sample{{5000, 100, 0, 1}, {5100, 200, 1, 1}, {9000, 300, 2, 1}}
	if len(got) != len(want) {
		t.Fatalf("got %d amostras", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("amostra %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
