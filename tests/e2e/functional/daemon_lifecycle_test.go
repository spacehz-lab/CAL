package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
)

func TestCALDaemonLifecycleCommands(t *testing.T) {
	repo, calctlBin, _ := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	env := e2etest.WithHomeEnv(os.Environ(), home)
	t.Cleanup(func() {
		_, _, _ = e2etest.RunCommand(repo, env, calctlBin, "daemon", "stop", "--json")
	})

	var status struct {
		Running bool   `json:"running"`
		Mode    string `json:"mode"`
	}
	e2etest.RunJSON(t, repo, env, &status, calctlBin, "daemon", "status", "--json")
	if status.Running || status.Mode != "local" {
		t.Fatalf("initial daemon status = %#v, want stopped local status", status)
	}

	var started struct {
		Running   bool   `json:"running"`
		Mode      string `json:"mode"`
		PID       int    `json:"pid"`
		Endpoint  string `json:"endpoint"`
		StartedAt string `json:"started_at"`
	}
	e2etest.RunJSON(t, repo, env, &started, calctlBin, "daemon", "start", "--json")
	if !started.Running || started.Mode != "local" || started.PID == 0 || started.Endpoint == "" || started.StartedAt == "" {
		t.Fatalf("daemon start = %#v, want running service status", started)
	}

	e2etest.RunJSON(t, repo, env, &status, calctlBin, "daemon", "status", "--json")
	if !status.Running || status.Mode != "local" {
		t.Fatalf("running daemon status = %#v, want running local status", status)
	}

	var repeated struct {
		Running bool `json:"running"`
		PID     int  `json:"pid"`
	}
	e2etest.RunJSON(t, repo, env, &repeated, calctlBin, "daemon", "start", "--json")
	if !repeated.Running || repeated.PID != started.PID {
		t.Fatalf("repeated daemon start = %#v, want existing running service pid %d", repeated, started.PID)
	}

	var stopped struct {
		Stopping bool `json:"stopping"`
	}
	e2etest.RunJSON(t, repo, env, &stopped, calctlBin, "daemon", "stop", "--json")
	if !stopped.Stopping {
		t.Fatalf("daemon stop = %#v, want stopping response", stopped)
	}
	waitForStoppedDaemon(t, repo, env, calctlBin)
}

func waitForStoppedDaemon(t *testing.T, repo string, env []string, calctlBin string) {
	t.Helper()
	var status struct {
		Running bool `json:"running"`
	}
	for attempt := 0; attempt < 100; attempt++ {
		e2etest.RunJSON(t, repo, env, &status, calctlBin, "daemon", "status", "--json")
		if !status.Running {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("daemon status still running after stop")
}
