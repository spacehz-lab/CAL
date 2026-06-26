package cald

import (
	"bytes"
	"testing"
)

func TestNewCommandDefinesServeCommand(t *testing.T) {
	var out bytes.Buffer
	cmd := NewCommand(CommandConfig{Out: &out, Err: &out, Home: t.TempDir()})
	serve, _, err := cmd.Find([]string{"serve"})
	if err != nil {
		t.Fatalf("Find(serve) error = %v", err)
	}
	if serve == nil || serve.Use != "serve" {
		t.Fatalf("serve command = %#v, want serve subcommand", serve)
	}
}
