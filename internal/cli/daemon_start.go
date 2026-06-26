package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald"
	caldclient "github.com/spacehz-lab/cal/internal/cald/client"
	"github.com/spacehz-lab/cal/internal/calpath"
	"github.com/spacehz-lab/cal/internal/logging"
)

const daemonStartTimeout = 10 * time.Second

type daemonStarter struct {
	cfg      Config
	caldPath string
	timeout  time.Duration
}

func newDaemonStartCommand(cfg Config) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the local CAL service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, err := newDaemonStarter(cfg).Start(cmd.Context())
			if err != nil {
				return writeCommandError(cmd, jsonOut, err)
			}
			if jsonOut {
				return writeJSON(cmd.OutOrStdout(), status)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "cald running")
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render machine-readable JSON")
	return cmd
}

func newDaemonStarter(cfg Config) daemonStarter {
	return daemonStarter{cfg: cfg, timeout: daemonStartTimeout}
}

func (starter daemonStarter) Start(ctx context.Context) (cald.Status, error) {
	home, err := starter.home()
	if err != nil {
		return cald.Status{}, err
	}
	if status, ok := starter.runningStatus(ctx, home); ok {
		return status, nil
	}
	_ = os.Remove(cald.EndpointFilePath(home))

	caldPath, err := starter.resolveCaldPath()
	if err != nil {
		return cald.Status{}, err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return cald.Status{}, fmt.Errorf("create CAL home: %w", err)
	}
	logFile, err := starter.openLog()
	if err != nil {
		return cald.Status{}, err
	}
	defer logFile.Close()

	cmd := exec.Command(caldPath, "serve")
	cmd.Env = calpath.WithHomeEnv(os.Environ(), home)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return cald.Status{}, newCommandErrorf(commandErrorCaldStartFailed, "start cald: %v", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, starter.startTimeout())
	defer cancel()
	status, err := starter.waitReady(waitCtx, home)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return cald.Status{}, newCommandErrorf(commandErrorCaldStartFailed, "cald did not become ready: %v", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return cald.Status{}, fmt.Errorf("release cald process: %w", err)
	}
	return status, nil
}

func (starter daemonStarter) home() (string, error) {
	home := strings.TrimSpace(starter.cfg.Home)
	if home != "" {
		return filepath.Clean(home), nil
	}
	return calpath.HomeDir()
}

func (starter daemonStarter) runningStatus(ctx context.Context, home string) (cald.Status, bool) {
	client, err := caldclient.New(home)
	if err != nil {
		return cald.Status{}, false
	}
	status, err := client.Status(ctx)
	return status, err == nil && status.Running
}

func (starter daemonStarter) resolveCaldPath() (string, error) {
	if strings.TrimSpace(starter.caldPath) != "" {
		return filepath.Clean(starter.caldPath), nil
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), caldExecutableName())
		if isFile(candidate) {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath("cald"); err == nil {
		return path, nil
	}
	return "", newCommandError(commandErrorCaldStartFailed, "cald executable was not found")
}

func (starter daemonStarter) openLog() (io.WriteCloser, error) {
	path, err := logging.ProcessLogPath("cald-daemon")
	if err != nil {
		return nil, fmt.Errorf("resolve cald daemon log path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open cald daemon log: %w", err)
	}
	return file, nil
}

func (starter daemonStarter) waitReady(ctx context.Context, home string) (cald.Status, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		client, err := caldclient.New(home)
		if err == nil {
			status, err := client.Status(ctx)
			if err == nil && status.Running {
				return status, nil
			}
			lastErr = err
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return cald.Status{}, lastErr
			}
			return cald.Status{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (starter daemonStarter) startTimeout() time.Duration {
	if starter.timeout > 0 {
		return starter.timeout
	}
	return daemonStartTimeout
}

func caldExecutableName() string {
	if runtime.GOOS == "windows" {
		return "cald.exe"
	}
	return "cald"
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
