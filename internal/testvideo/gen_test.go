package testvideo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWrite gera o vídeo numa pasta temporária, ou em $TESTVIDEO_OUT, se
// definido (para experimentar a interface):
//
//	TESTVIDEO_OUT=/tmp/GH010001.MP4 go test ./internal/testvideo -run TestWrite
func TestWrite(t *testing.T) {
	p := os.Getenv("TESTVIDEO_OUT")
	opt := Options{}
	if p == "" {
		p = filepath.Join(t.TempDir(), "GH010001.MP4")
	} else {
		opt.Filler = 1 << 30 // 1 GiB, para ver o progresso e o botão de abortar
	}
	if err := Write(p, opt); err != nil {
		t.Fatal(err)
	}
}
