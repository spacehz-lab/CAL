package cald

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/cald/endpoint"
)

func TestNewCommandBuildsServeOnly(t *testing.T) {
	cmd, err := NewCommand(CommandOptions{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Environ: []string{}})
	if err != nil {
		t.Fatalf("NewCommand() error = %v", err)
	}
	if cmd.Name() != commandName {
		t.Fatalf("command name = %q, want %s", cmd.Name(), commandName)
	}
	if len(cmd.Commands()) != 1 || cmd.Commands()[0].Name() != "serve" {
		t.Fatalf("commands = %#v, want serve only", cmd.Commands())
	}
}

func TestNewCommandResolvesHomeFromEnv(t *testing.T) {
	home := t.TempDir()
	cmd, err := NewCommand(CommandOptions{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Environ: []string{envHome + "=" + home}})
	if err != nil {
		t.Fatalf("NewCommand() error = %v", err)
	}
	flag := cmd.PersistentFlags().Lookup(flagHome)
	if flag == nil || flag.Value.String() != home {
		t.Fatalf("home flag = %#v, want %s", flag, home)
	}
}

func TestServeCommandUsesHomeFlag(t *testing.T) {
	initialHome := t.TempDir()
	flagHomeValue := t.TempDir()
	cmd, err := NewCommand(CommandOptions{Home: initialHome, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Environ: []string{}})
	if err != nil {
		t.Fatalf("NewCommand() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		cmd.SetArgs([]string{"--home", flagHomeValue, "serve"})
		errCh <- cmd.ExecuteContext(ctx)
	}()

	waitForEndpoint(t, flagHomeValue)
	cancel()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("ExecuteContext() error = %v, want context.Canceled", err)
	}
	assertNoEndpoint(t, initialHome)
	assertNoEndpoint(t, flagHomeValue)
}

func waitForEndpoint(t *testing.T, home string) endpoint.Record {
	t.Helper()
	var lastErr error
	for attempt := 0; attempt < 100; attempt++ {
		record, ok, err := endpoint.Read(home)
		if err == nil && ok {
			return record
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("endpoint was not written, last error = %v", lastErr)
	return endpoint.Record{}
}

func assertNoEndpoint(t *testing.T, home string) {
	t.Helper()
	if _, ok, err := endpoint.Read(home); err != nil || ok {
		t.Fatalf("endpoint read for %s = ok:%v err:%v, want missing", home, ok, err)
	}
}
