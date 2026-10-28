BIN := bin/gopro-gps-repair

# Versão: vem da tag do Git (vX.Y.Z). Depois da tag, X.Y.Z-N-gHASH; sem tag,
# 0.0.0-dev. Gravada no binário pelo pacote internal/versao.
VERSION ?= $(or $(patsubst v%,%,$(shell git describe --tags --match 'v[0-9]*' 2>/dev/null)),0.0.0-dev)
LDFLAGS := -X gopro-gps-repair/internal/versao.Version=$(VERSION)

.PHONY: build versao test vet diagnosticar reparar gui gui-dev pacotes conferir-versao flatpak fontes-flatpak clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/gopro-gps-repair

# Mostra a versão que o build vai usar
versao:
	@echo $(VERSION)

test:
	go test ./...

vet:
	go vet ./...

# make diagnosticar VIDEO=~/Downloads/GH019736.MP4
diagnosticar: build
	$(BIN) diagnosticar $(VIDEO) -blocos

# make reparar VIDEO=~/Downloads/GH019736.MP4 [ARGS="-gpx trilha.gpx"]
reparar: build
	$(BIN) reparar $(VIDEO) $(ARGS)

# Interface gráfica (Wails v3 + React): gera gui/bin/gopro-gps-repair-gui
gui:
	cd gui && wails3 build

# Interface em modo de desenvolvimento, com recarga automática
gui-dev:
	cd gui && wails3 dev -config ./build/config.yml

# Pacotes Linux (.deb, .rpm, Arch e AppImage) em gui/bin, com a versão da tag
pacotes:
	cd gui && wails3 task linux:package VERSION=$(VERSION)

# Antes de criar a tag vX.Y.Z: as lojas exigem a versão escrita no metainfo
# (<release>) e no AUR (pkgver). Confere as duas contra TAG (padrão: a última
# tag). Ex.: make conferir-versao TAG=v0.3.0
METAINFO := gui/build/linux/share/io.github.PauloTorresApps.gopro_gps_repair.metainfo.xml
TAG ?= $(shell git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null)
conferir-versao:
	@[ -n "$(TAG)" ] || { echo "nenhuma tag vX.Y.Z; use: make conferir-versao TAG=vX.Y.Z"; exit 1; }
	@v=$(patsubst v%,%,$(TAG)); erro=0; \
	grep -q "<release version=\"$$v\"" $(METAINFO) || { echo "$(METAINFO): falta <release version=\"$$v\">"; erro=1; }; \
	grep -qx "pkgver=$$v" packaging/aur/PKGBUILD || { echo "packaging/aur/PKGBUILD: pkgver não é $$v"; erro=1; }; \
	grep -qx "	pkgver = $$v" packaging/aur/.SRCINFO || { echo "packaging/aur/.SRCINFO: pkgver não é $$v"; erro=1; }; \
	[ $$erro = 0 ] && echo "versão $$v: metainfo e AUR conferem"

# Flatpak local (requer org.flatpak.Builder e o SDK do GNOME instalados)
flatpak:
	flatpak run --filesystem=$(CURDIR) org.flatpak.Builder --user --install --force-clean \
		build-flatpak packaging/flatpak/io.github.PauloTorresApps.gopro_gps_repair.yml

# Regera as listas de dependências do Flatpak depois de mudar go.mod ou package-lock.json
fontes-flatpak:
	cd gui && go mod download
	python3 packaging/flatpak/gen-go-sources.py > packaging/flatpak/go-sources.json
	python3 packaging/flatpak/gen-npm-sources.py > packaging/flatpak/npm-sources.json

clean:
	rm -rf bin gui/bin build-flatpak .flatpak-builder
