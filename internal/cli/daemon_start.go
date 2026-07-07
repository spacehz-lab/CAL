package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/cli/client"
	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/logging"
)

const (
	daemonBinaryName   = "cald"
	daemonLogName      = "cald-daemon"
	daemonServeCommand = "serve"

	defaultDaemonStartTimeout = 10 * time.Second
	daemonPollInterval        = 100 * time.Millisecond
)

type daemonStarter struct {
	home    string
	env     []string
	path    string
	logPath string
	timeout time.Duration
}

func (cli *CLI) newDaemonStarter() *daemonStarter {
	return &daemonStarter{
		home:    cli.home,
		env:     cli.env,
		timeout: defaultDaemonStartTimeout,
	}
}

func (starter *daemonStarter) Start(ctx context.Context) (*contract.DaemonStatus, error) {
	if starter == nil {
		return nil, fmt.Errorf("daemon starter is required")
	}
	home, err := resolveHome(starter.home, starter.env)
	if err != nil {
		return nil, err
	}
	if status, ok := starter.runningStatus(ctx, home); ok {
		return status, nil
	}

	path, err := starter.resolvePath()
	if err != nil {
		return nil, err
	}
	logFile, err := starter.openLog()
	if err != nil {
		return nil, err
	}
	defer logFile.Close()

	cmd := exec.Command(path, "--"+flagHome, home, daemonServeCommand)
	cmd.Env = envWithHome(starter.env, home)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start cald: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, starter.startTimeout())
	defer cancel()
	status, err := starter.waitReady(waitCtx, home)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	if err := cmd.Process.Release(); err != nil {
		return nil, fmt.Errorf("release cald process: %w", err)
	}
	return status, nil
}

func (starter *daemonStarter) runningStatus(ctx context.Context, home string) (*contract.DaemonStatus, bool) {
	status, err := starter.status(ctx, home)
	return status, err == nil && status != nil && status.Running
}

func (starter *daemonStarter) waitReady(ctx context.Context, home string) (*contract.DaemonStatus, error) {
	ticker := time.NewTicker(daemonPollInterval)
	defer ticker.Stop()

	for {
		if status, ok := starter.runningStatus(ctx, home); ok {
			return status, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for cald: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (starter *daemonStarter) status(ctx context.Context, home string) (*contract.DaemonStatus, error) {
	daemonClient, err := client.New(client.Options{Home: home})
	if err != nil {
		return nil, err
	}
	return daemonClient.Status(ctx)
}

func (starter *daemonStarter) resolvePath() (string, error) {
	if path := strings.TrimSpace(starter.path); path != "" {
		return filepath.Clean(path), nil
	}
	if path, ok := siblingDaemonPath(); ok {
		return path, nil
	}
	path, err := exec.LookPath(daemonExecutableName())
	if err != nil {
		return "", errors.New("cald executable was not found")
	}
	return path, nil
}

func (starter *daemonStarter) openLog() (*os.File, error) {
	path := strings.TrimSpace(starter.logPath)
	if path == "" {
		var err error
		path, err = logging.Path(daemonLogName)
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open daemon log: %w", err)
	}
	return file, nil
}

func (starter *daemonStarter) startTimeout() time.Duration {
	if starter.timeout > 0 {
		return starter.timeout
	}
	return defaultDaemonStartTimeout
}

func siblingDaemonPath() (string, bool) {
	current, err := os.Executable()
	if err != nil {
		return "", false
	}
	path := filepath.Join(filepath.Dir(current), daemonExecutableName())
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path, true
	}
	return "", false
}

func daemonExecutableName() string {
	if runtime.GOOS == "windows" {
		return daemonBinaryName + ".exe"
	}
	return daemonBinaryName
}

func envWithHome(env []string, home string) []string {
	if env == nil {
		env = os.Environ()
	}
	next := make([]string, 0, len(env)+1)
	replaced := false
	for _, item := range env {
		if strings.HasPrefix(item, envHome+"=") {
			next = append(next, envHome+"="+home)
			replaced = true
			continue
		}
		next = append(next, item)
	}
	if !replaced {
		next = append(next, envHome+"="+home)
	}
	return next
}
