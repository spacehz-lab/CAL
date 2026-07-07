package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/pkg/jsonfile"
)

const (
	verifyParamFormat = "format"
	verifyParamValue  = "value"
)

type localCLIWorkspace struct {
	repo     string
	home     string
	calctl   string
	cald     string
	provider model.Provider
	env      []string
}

type daemonEndpoint struct {
	BaseURL string `json:"base_url"`
}

type providerListResponse struct {
	Providers []model.Provider `json:"providers"`
}

type runResponse struct {
	Run *model.Run `json:"run,omitempty"`
}

type useResponse struct {
	Selection *struct {
		CapabilityID string `json:"capability_id"`
		BindingID    string `json:"binding_id"`
		ProviderID   string `json:"provider_id"`
	} `json:"selection,omitempty"`
	Run    *model.Run      `json:"run,omitempty"`
	Status model.RunStatus `json:"status"`
}

type evalResponse struct {
	Acquisition struct {
		Traces struct {
			Total int `json:"total"`
		} `json:"traces"`
	} `json:"acquisition"`
	Reuse struct {
		Runs struct {
			Total  int            `json:"total"`
			ByName map[string]int `json:"by_name,omitempty"`
		} `json:"runs"`
		Verified int `json:"verified"`
	} `json:"reuse"`
	Capability struct {
		Capabilities       int `json:"capabilities"`
		Bindings           int `json:"bindings"`
		PromotedBindings   int `json:"promoted_bindings"`
		BindingsWithVerify int `json:"bindings_with_verify"`
	} `json:"capability"`
}

func newLocalCLIWorkspace(t *testing.T, providerPath string) localCLIWorkspace {
	t.Helper()
	temp := t.TempDir()
	repo := repoRoot(t)
	workspace := localCLIWorkspace{
		repo:   repo,
		home:   filepath.Join(temp, "home"),
		calctl: filepath.Join(temp, "calctl"),
		cald:   filepath.Join(temp, "cald"),
	}
	buildGoPackage(t, repo, workspace.calctl, "./cmd/calctl")
	buildGoPackage(t, repo, workspace.cald, "./cmd/cald")
	workspace.env = withHomeEnv(os.Environ(), workspace.home)
	startCald(t, workspace.repo, workspace.env, workspace.cald)
	workspace.provider = addProvider(t, workspace.repo, workspace.env, workspace.calctl, providerPath)
	return workspace
}

func (workspace localCLIWorkspace) runJSON(t *testing.T, target any, args ...string) {
	t.Helper()
	runJSON(t, workspace.repo, workspace.env, target, workspace.calctl, args...)
}

func (workspace localCLIWorkspace) seedCapability(t *testing.T, capabilityID string, description string, execution model.Execution, verify *model.VerifySpec) model.Capability {
	t.Helper()
	bindingID, err := model.BindingIDForExecution(capabilityID, workspace.provider.ID, execution)
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	capability := model.Capability{
		ID:          capabilityID,
		Description: description,
		Bindings: []model.Binding{{
			ID:           bindingID,
			CapabilityID: capabilityID,
			ProviderID:   workspace.provider.ID,
			Execution:    execution,
			Verify:       verify,
			Evidence:     []model.EvidenceRef{{ID: "local_cli_seed", Type: "fixture"}},
			State:        model.BindingStatePromoted,
		}},
	}
	if err := os.MkdirAll(filepath.Join(workspace.home, "capabilities"), 0o755); err != nil {
		t.Fatalf("create capabilities dir: %v", err)
	}
	if err := jsonfile.WriteAtomic(filepath.Join(workspace.home, "capabilities", capabilityID+".json"), &capability); err != nil {
		t.Fatalf("write capability: %v", err)
	}
	return capability
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

func buildGoPackage(t *testing.T, repo string, output string, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", output, pkg)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build %s failed: %v\n%s", pkg, err, out)
	}
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

func writePNG(t *testing.T, path string, width int, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{G: 0xff, A: 0xff})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create PNG: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
}

func assertPNGDimensions(t *testing.T, path string, width int, height int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open PNG: %v", err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode PNG config: %v", err)
	}
	if config.Width != width || config.Height != height {
		t.Fatalf("PNG dimensions = %dx%d, want %dx%d", config.Width, config.Height, width, height)
	}
}

func assertSuccessfulVerifiedRun(t *testing.T, run *model.Run, capabilityID string, providerID string) {
	t.Helper()
	if run == nil {
		t.Fatalf("run = nil, want successful %s run", capabilityID)
	}
	if run.Status != model.RunStatusSucceeded || !run.Verified || run.CapabilityID != capabilityID || run.ProviderID != providerID {
		t.Fatalf("run = %#v, want verified successful %s run on provider %s", run, capabilityID, providerID)
	}
	if len(run.Evidence) == 0 {
		t.Fatalf("run evidence = %#v, want verification evidence", run.Evidence)
	}
}

func assertEval(t *testing.T, metrics evalResponse, wantRuns int, wantVerified int) {
	t.Helper()
	if metrics.Capability.Capabilities != 1 || metrics.Capability.Bindings != 1 || metrics.Capability.PromotedBindings != 1 || metrics.Capability.BindingsWithVerify != 1 {
		t.Fatalf("eval capability = %#v, want one promoted verified capability", metrics.Capability)
	}
	if metrics.Reuse.Runs.Total != wantRuns || metrics.Reuse.Runs.ByName[string(model.RunStatusSucceeded)] != wantRuns || metrics.Reuse.Verified != wantVerified {
		t.Fatalf("eval reuse = %#v, want %d successful runs and %d verified", metrics.Reuse, wantRuns, wantVerified)
	}
	if metrics.Acquisition.Traces.Total != 0 {
		t.Fatalf("eval acquisition = %#v, want no acquisition traces for seeded local CLI fixture", metrics.Acquisition)
	}
}

func jsonInputs(values map[string]any) string {
	content, err := json.Marshal(values)
	if err != nil {
		panic(fmt.Sprintf("marshal JSON inputs: %v", err))
	}
	return string(content)
}
