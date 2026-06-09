BINARY := keys7
PKG := ./cmd/keys7

.PHONY: build run-mock build-windows vet test clean

# Default build: mock source, pure Go, no CGO.
build:
	go build -o bin/$(BINARY) $(PKG)

# Run with the synthetic MIDI source (works anywhere, incl. WSL).
run-mock:
	go run $(PKG) --source=mock

# Device-capable Windows build. Intended to run ON the Windows host (the P-125
# is USB-MIDI on Windows; WSL doesn't see it). Needs CGO + a C toolchain + RtMidi.
build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -tags midi_device -o bin/$(BINARY).exe $(PKG)

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin
