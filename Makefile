BINARY := keys7
PKG := ./cmd/keys7

.PHONY: build run-mock build-windows build-play7 build-play7-windows build-export7 install-windows vet test clean

# Windows deploy dir. Auto-detects the Windows user profile (no hard-coded user,
# so this stays generic); override e.g. make install-windows WINDEST=/mnt/d/keys7.
WINDEST ?= $(shell wslpath "$$(cmd.exe /c 'echo %USERPROFILE%' 2>/dev/null | tr -d '\r')" 2>/dev/null)/Documents/keys7

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

# export7: turn a session journal into a .mid (bridge back to Renoise/MuseScore).
# Runs in WSL, reading the journal from the Windows side under /mnt/c.
build-export7:
	go build -o bin/export7 ./cmd/export7

# One-shot deploy from WSL: cross-compile both binaries, copy them and the
# launcher to the Windows side, and drop a Desktop shortcut (Windows Terminal).
# After this, launching a session is a double-click — no PowerShell, no cd.
install-windows: build-windows build-play7-windows
	@mkdir -p "$(WINDEST)/sessions"
	@# Hot-swap: a running .exe can't be overwritten on Windows, but it CAN be
	@# renamed — so move any in-use binary aside, then copy. The stale .old is
	@# deleted if free, else left for the live session to release when it exits.
	@for b in keys7.exe play7.exe; do \
	  [ -f "$(WINDEST)/$$b" ] && mv -f "$(WINDEST)/$$b" "$(WINDEST)/$$b.old" 2>/dev/null || true; \
	  cp "bin/$$b" "$(WINDEST)/$$b"; \
	  rm -f "$(WINDEST)/$$b.old" 2>/dev/null || true; \
	done
	cp scripts/keys7.ps1 "$(WINDEST)/"
	@echo "binaries + launcher -> $(WINDEST)"
	powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$$(wslpath -w scripts/install-shortcut.ps1)" -InstallDir "$$(wslpath -w '$(WINDEST)')"

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin
