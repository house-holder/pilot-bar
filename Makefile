PREFIX := $(HOME)/.local/bin

build:
	go build -o $(PREFIX)/pilot-bar-daemon ./cmd/daemon/.
	go build -o $(PREFIX)/pilot-bar ./cmd/waybar/.

dev: build
	pilot-bar-daemon switch $(or $(ICAO),KCGI)
