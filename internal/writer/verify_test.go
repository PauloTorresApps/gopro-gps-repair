package writer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyOnlyPatched(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "orig")
	out := filepath.Join(dir, "out")
	data := make([]byte, 3<<20) // maior que o bloco de leitura
	for i := range data {
		data[i] = byte(i)
	}
	if err := os.WriteFile(orig, data, 0o644); err != nil {
		t.Fatal(err)
	}
	patches := []Patch{{Off: 100, Len: 8}, {Off: 2<<20 + 5, Len: 4}}

	write := func(mod func([]byte)) {
		d := append([]byte(nil), data...)
		mod(d)
		if err := os.WriteFile(out, d, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(func(d []byte) { d[100]++; d[107]++; d[2<<20+8]++ })
	if err := VerifyOnlyPatched(orig, out, patches); err != nil {
		t.Fatalf("mudança dentro dos trechos deveria passar: %v", err)
	}

	write(func(d []byte) { d[108]++ }) // um byte depois do 1º trecho
	if err := VerifyOnlyPatched(orig, out, patches); err == nil {
		t.Fatal("mudança fora dos trechos deveria falhar")
	}

	if err := os.WriteFile(out, data[:len(data)-1], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyOnlyPatched(orig, out, patches); err == nil {
		t.Fatal("tamanho diferente deveria falhar")
	}
}
