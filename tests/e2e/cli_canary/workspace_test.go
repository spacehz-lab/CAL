package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

const (
	envCliCanaryE2E      = "CAL_CLI_CANARY_E2E"
	envCliCanaryKeepHome = "CAL_CLI_CANARY_KEEP_HOME"
	envLLMAPI            = "CAL_LLM_API"
	envLLMModel          = "CAL_LLM_MODEL"
	envLLMAPIKey         = "CAL_LLM_API_KEY"
)

type cliCanaryWorkspace struct {
	root string
	home string
	temp string
}

type daemonEndpoint struct {
	BaseURL string `json:"base_url"`
}

type providerListResponse struct {
	Providers []model.Provider `json:"providers"`
}

type streamEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

func newCliCanaryWorkspace(t *testing.T) cliCanaryWorkspace {
	t.Helper()
	root, err := os.MkdirTemp("", t.Name())
	if err != nil {
		t.Fatalf("create CLI canary workspace: %v", err)
	}
	workspace := cliCanaryWorkspace{
		root: root,
		home: filepath.Join(root, "home"),
		temp: filepath.Join(root, "work"),
	}
	if err := os.MkdirAll(workspace.temp, 0o755); err != nil {
		t.Fatalf("create CLI canary work dir: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() || os.Getenv(envCliCanaryKeepHome) == "1" {
			logCliCanaryWorkspace(t, workspace)
			return
		}
		_ = os.RemoveAll(root)
	})
	return workspace
}

func cliCanaryEnv(t *testing.T, home string) []string {
	t.Helper()
	if !cliCanaryEnabled() {
		t.Skip("set CAL_CLI_CANARY_E2E=1 and CAL_LLM_* to run CLI canary e2e")
	}
	required := []string{envLLMAPI, envLLMModel, envLLMAPIKey}
	for _, name := range required {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			t.Skipf("set %s to run CLI canary e2e", name)
		}
	}
	if os.Getenv(envLLMAPI) != "chat_completions" {
		t.Skip("CLI canary e2e requires CAL_LLM_API=chat_completions")
	}
	return withHomeEnv(os.Environ(), home)
}

func cliCanaryEnabled() bool {
	return os.Getenv(envCliCanaryE2E) == "1"
}

func addProvider(t *testing.T, repo string, env []string, calctlBin string, providerPath string) model.Provider {
	t.Helper()
	var response providerListResponse
	runJSON(t, repo, env, &response, calctlBin, "providers", "add", "--provider-path", providerPath, "--json")
	for _, provider := range response.Providers {
		if provider.Path == providerPath {
			return provider
		}
	}
	t.Fatalf("provider path %s not found in response %#v", providerPath, response)
	return model.Provider{}
}

func runAcquisition(t *testing.T, repo string, env []string, calctlBin string, providerID string, hint string) contract.AcquisitionResponse {
	t.Helper()
	var response contract.AcquisitionResponse
	command := []string{"acquisition", "run", "--provider-id", providerID, "--hint", hint, "--stream", "--json"}
	events := runJSONStream(t, repo, env, &response, calctlBin, command...)
	if response.Error != nil || response.CapabilitiesPromoted == 0 {
		t.Logf("CLI canary acquisition stream events:\n%s", formatStreamEvents(events))
	}
	return response
}

func runJSON(t *testing.T, repo string, env []string, target any, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %s failed: %v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), err, stderr.String(), out)
	}
	if err := json.Unmarshal(out, target); err != nil {
		t.Fatalf("%s %s returned invalid JSON: %v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), err, stderr.String(), out)
	}
}

func runJSONStream(t *testing.T, repo string, env []string, target any, name string, args ...string) []streamEnvelope {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %s failed: %v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), err, stderr.String(), out)
	}
	events := decodeStreamEvents(t, out, name, args...)
	var sawResult bool
	for _, event := range events {
		switch event.Event {
		case "result":
			if err := json.Unmarshal(event.Data, target); err != nil {
				t.Fatalf("%s %s returned invalid stream result: %v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), err, stderr.String(), out)
			}
			sawResult = true
		case "error":
			var response contract.ErrorResponse
			if err := json.Unmarshal(event.Data, &response); err != nil {
				t.Fatalf("%s %s returned invalid stream error: %v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), err, stderr.String(), out)
			}
			t.Fatalf("%s %s returned stream error: %#v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), response, stderr.String(), out)
		}
	}
	if !sawResult {
		t.Fatalf("%s %s returned stream without result\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), stderr.String(), out)
	}
	return events
}

func startCald(t *testing.T, repo string, env []string, caldBin string) {
	t.Helper()
	cmd := exec.Command(caldBin, "serve")
	cmd.Dir = repo
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start cald: %v\n%s", err, stderr.String())
	}
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
		}
	})
	waitForCaldEndpoint(t, env, &stderr, done, func() error { return waitErr })
}

func decodeStreamEvents(t *testing.T, out []byte, name string, args ...string) []streamEnvelope {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	events := make([]streamEnvelope, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event streamEnvelope
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("%s %s returned invalid JSONL event: %v\nline=%s\nstdout=%s", name, strings.Join(args, " "), err, line, out)
		}
		events = append(events, event)
	}
	return events
}

func formatStreamEvents(events []streamEnvelope) string {
	var b strings.Builder
	for _, event := range events {
		b.WriteString(event.Event)
		if len(event.Data) > 0 {
			b.WriteByte(' ')
			b.Write(event.Data)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func waitForCaldEndpoint(t *testing.T, env []string, stderr *bytes.Buffer, done <-chan struct{}, waitErr func() error) {
	t.Helper()
	home := homeFromEnv(env)
	if home == "" {
		t.Fatal("CAL home is required to wait for cald endpoint")
	}
	endpointPath := filepath.Join(home, "cald", "endpoint.json")
	client := http.Client{Timeout: time.Second}
	var lastErr error
	started := time.Now()
	deadline := started.Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			t.Fatalf("cald exited before endpoint became ready after %s: %v\nendpoint=%s\nstderr=%s", time.Since(started).Round(time.Millisecond), waitErr(), endpointPath, stderr.String())
		default:
		}
		content, err := os.ReadFile(endpointPath)
		if err == nil {
			var endpoint daemonEndpoint
			if err := json.Unmarshal(content, &endpoint); err == nil && endpoint.BaseURL != "" {
				resp, err := client.Get(strings.TrimRight(endpoint.BaseURL, "/") + "/v1/daemon/status")
				if err == nil {
					_ = resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						return
					}
				}
				lastErr = err
			} else {
				lastErr = err
			}
		} else {
			lastErr = err
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("cald endpoint did not become ready after %s: %v\nendpoint=%s\nstderr=%s", time.Since(started).Round(time.Millisecond), lastErr, endpointPath, stderr.String())
}

func homeFromEnv(env []string) string {
	const key = "CAL_HOME="
	for _, item := range env {
		if strings.HasPrefix(item, key) {
			return item[len(key):]
		}
	}
	return ""
}

func withHomeEnv(env []string, home string) []string {
	const key = "CAL_HOME="
	value := key + filepath.Clean(home)
	for index, item := range env {
		if strings.HasPrefix(item, key) {
			env[index] = value
			return env
		}
	}
	return append(env, value)
}
