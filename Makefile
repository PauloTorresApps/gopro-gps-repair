BIN := bin/gopro-gps-repair

.PHONY: build test vet diagnosticar reparar gui gui-dev clean

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

clean:
	rm -rf bin gui/bin
