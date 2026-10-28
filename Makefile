BIN := bin/gopro-gps-repair

.PHONY: build test vet diagnosticar reparar gui gui-dev pacotes flatpak fontes-flatpak clean

build:
	go build -o $(BIN) ./cmd/gopro-gps-repair

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

# Pacotes Linux (.deb, .rpm, Arch e AppImage) em gui/bin; VERSION opcional
pacotes:
	cd gui && wails3 task linux:package $(if $(VERSION),VERSION=$(VERSION))

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
