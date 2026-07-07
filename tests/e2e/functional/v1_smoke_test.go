package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestV1CLISmokeAndDaemonLifecycle(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()
	home := filepath.Join(temp, "home")
	env := withHomeEnv(os.Environ(), home)

	if output := runCombined(t, repo, env, calctlBin, "--help"); !strings.Contains(output, "Capability Acquisition Loop") {
		t.Fatalf("calctl --help output missing title:\n%s", output)
	}
	if output := runCombined(t, repo, env, caldBin, "--help"); !strings.Contains(output, "CAL local daemon") {
		t.Fatalf("cald --help output missing title:\n%s", output)
	}
	if output := runCombined(t, repo, env, caldBin, "serve", "--help"); !strings.Contains(output, "Start the local CAL daemon") {
		t.Fatalf("cald serve --help output missing title:\n%s", output)
	}

	var stopped daemonStatus
	runJSON(t, repo, env, &stopped, calctlBin, "daemon", "status", "--json")
	if stopped.Running {
		t.Fatalf("initial daemon status = %#v, want stopped", stopped)
	}

	var running daemonStatus
	runJSON(t, repo, env, &running, calctlBin, "daemon", "start", "--json")
	if !running.Running || running.BaseURL == "" || running.PID == 0 {
		t.Fatalf("running daemon status = %#v, want running endpoint", running)
	}

	var stoppedResponse struct {
		Stopping bool `json:"stopping"`
	}
	runJSON(t, repo, env, &stoppedResponse, calctlBin, "daemon", "stop", "--json")
	if !stoppedResponse.Stopping {
		t.Fatalf("daemon stop = %#v, want stopping", stoppedResponse)
	}
	waitForStoppedDaemon(t, repo, env, calctlBin)
}

func waitForStoppedDaemon(t *testing.T, repo string, env []string, calctlBin string) {
	t.Helper()
	for attempt := 0; attempt < 100; attempt++ {
		var status daemonStatus
		runJSON(t, repo, env, &status, calctlBin, "daemon", "status", "--json")
		if !status.Running {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("daemon status still running after stop")
}
