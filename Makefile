BINDIR ?= build/bin

.PHONY: build install test e2e clean

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/calctl ./cmd/calctl
	go build -o $(BINDIR)/cald ./cmd/cald

install:
	go install ./cmd/calctl ./cmd/cald

test:
	go test ./... -count=1 -p 1

e2e:
	go test ./tests/e2e/functional -count=1

clean:
	rm -rf build/bin
