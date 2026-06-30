BINDIR ?= build/bin
LIVE_LLM_TEST_TIMEOUT ?= 20m
CLI_CANARY_LLM_TEST_TIMEOUT ?= 20m

.PHONY: build install test e2e e2e-live-llm e2e-cli-canary-llm clean

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

e2e-live-llm:
	go test ./tests/e2e/live_llm -count=1 -v -timeout $(LIVE_LLM_TEST_TIMEOUT)

e2e-cli-canary-llm:
	go test ./tests/e2e/cli_canary_llm -count=1 -v -timeout $(CLI_CANARY_LLM_TEST_TIMEOUT)

clean:
	rm -rf build/bin
