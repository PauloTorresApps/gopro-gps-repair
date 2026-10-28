// Pacote versao informa a versão do app. A fonte é a tag do Git (vX.Y.Z): o
// build a lê com `git describe` e a grava no binário com
//
//	-ldflags "-X gopro-gps-repair/internal/versao.Version=X.Y.Z"
//
// (Makefile, Taskfile, Flatpak, Snap e AUR já fazem isso).
package versao

import (
	"runtime/debug"
	"strings"
)

// Version é gravada na compilação; fica vazia num `go build` simples.
var Version string

// Atual devolve a versão do app. Sem -ldflags, usa a versão do módulo
// registrada pelo Go (instalação com `go install ...@vX.Y.Z`); sem nenhuma
// das duas, devolve "dev".
func Atual() string {
	if Version != "" {
		return strings.TrimPrefix(Version, "v")
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Path == "gopro-gps-repair" {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return "dev"
}
