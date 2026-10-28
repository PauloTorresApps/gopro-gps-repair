// Pacote goprogpsrepair guarda a versão do app, lida do arquivo VERSION (a
// fonte única) e embutida no binário na compilação. Assim qualquer build
// (Makefile, Wails, Flatpak, Snap, AUR) leva a versão certa sem -ldflags.
package goprogpsrepair

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version é a versão do app, no formato x.y.z.
var Version = strings.TrimSpace(versionFile)
