BINARY := keys7
PKG := ./cmd/keys7

.PHONY: build run-mock build-windows build-play7 build-play7-windows vet test clean

# Default build: mock source, pure Go, no CGO.
build:
	go build -o bin/$(BINARY) $(PKG)

# Run with the synthetic MIDI source (works anywhere, incl. WSL).
run-mock:
	go run $(PKG) --source=mock

# Windows build (the P-125 is USB-MIDI on Windows; WSL doesn't see it). The
# device source uses WinMM in pure Go, so this cross-compiles from WSL with no
# CGO and no toolchain — just copy bin/keys7.exe to Windows and run it.
build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(BINARY).exe $(PKG)

# play7: the output twin (plays sequences on the piano). Same cross-compile
# story as keys7: WinMM in pure Go, copy bin/play7.exe next to keys7.exe.
build-play7:
	go build -o bin/play7 ./cmd/play7

build-play7-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/play7.exe ./cmd/play7

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin
