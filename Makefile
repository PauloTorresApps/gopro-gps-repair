BIN := bin/gopro-gps-repair

.PHONY: build test vet diagnosticar reparar clean

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

clean:
	rm -rf bin
