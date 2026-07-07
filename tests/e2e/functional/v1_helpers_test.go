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

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/pkg/jsonfile"
)

type daemonStatus struct {
	Running bool   `json:"running"`
	BaseURL string `json:"base_url,omitempty"`
	PID     int    `json:"pid,omitempty"`
}

type providerListResponse struct {
	Providers []model.Provider `json:"providers"`
}

type capabilityListResponse struct {
	Count        int `json:"count"`
	Capabilities []struct {
		ID       string `json:"id"`
		Bindings struct {
			Available int `json:"available"`
		} `json:"bindings"`
	} `json:"capabilities"`
}

type runResponse struct {
	Run *model.Run `json:"run,omitempty"`
}

type acquisitionResponse struct {
	TraceID              string       `json:"trace_id"`
	CapabilitiesPromoted int          `json:"capabilities_promoted"`
	BindingsPromoted     int          `json:"bindings_promoted"`
	Trace                *model.Trace `json:"trace,omitempty"`
}

type useResponse struct {
	ID        string `json:"id"`
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
			Total  int            `json:"total"`
			ByName map[string]int `json:"by_name,omitempty"`
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

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func addV1Provider(t *testing.T, repo string, env []string, calctlBin string, providerPath string) model.Provider {
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

func runFailJSON(t *testing.T, repo string, env []string, target any, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	out, err := cmd.Output()
	if err == nil {
		t.Fatalf("%s %s succeeded, want failure\n%s", name, strings.Join(args, " "), out)
	}
	if err := json.Unmarshal(out, target); err != nil {
		t.Fatalf("%s %s returned invalid failure JSON: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

func runCombined(t *testing.T, repo string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFakeExecutable(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Fake Exporter"
  echo "CAL_CAPABILITY document.convert"
  echo "CAL_COMMAND export-pdf --source {{source}} --target {{target}}"
  echo "Usage: fake-exporter export-pdf --source PATH --target PATH"
  exit 0
fi
if [ "$1" = "export-pdf" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --source)
        source="$2"
        shift 2
        ;;
      --target)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$source" ] || [ -z "$target" ]; then
    exit 2
  fi
  printf '%s\n' '%PDF-1.4' '1 0 obj' '<< /Type /Catalog /Pages 2 0 R >>' 'endobj' '2 0 obj' '<< /Type /Pages /Kids [3 0 R] /Count 1 >>' 'endobj' '3 0 obj' '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R >>' 'endobj' '4 0 obj' '<< /Length 44 >>' 'stream' 'BT /F1 12 Tf 10 100 Td (fake pdf) Tj ET' 'endstream' 'endobj' 'xref' '0 5' '0000000000 65535 f ' '0000000009 00000 n ' '0000000058 00000 n ' '0000000115 00000 n ' '0000000202 00000 n ' 'trailer' '<< /Root 1 0 R /Size 5 >>' 'startxref' '295' '%%EOF' > "$target"
  cat "$target"
  exit 0
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
}

type caldEndpoint struct {
	BaseURL string `json:"base_url"`
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
			var endpoint caldEndpoint
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

func seedDocumentConvertCapability(t *testing.T, home string, provider model.Provider) model.Capability {
	t.Helper()
	execution := model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{
			model.ExecutionSpecArgs:            []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"},
			model.ExecutionSpecStdoutPathInput: "target",
		},
	}
	bindingID, err := model.BindingIDForExecution("document.convert", provider.ID, execution)
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	capability := model.Capability{
		ID:          "document.convert",
		Description: "Convert a document into PDF.",
		Bindings: []model.Binding{{
			ID:           bindingID,
			CapabilityID: "document.convert",
			ProviderID:   provider.ID,
			Execution:    execution,
			Verify: &model.VerifySpec{
				Level:  model.VerifyLevelL2,
				Method: model.VerifyMethodExecute,
				Checks: []model.VerifyCheck{{
					Subject:   model.VerifySubject{Type: model.VerifySubjectExitCode},
					Predicate: model.VerifyPredicateEquals,
					Params:    map[string]any{"value": 0},
				}},
			},
			Evidence: []model.EvidenceRef{{ID: "seed_evidence", Type: "fixture"}},
			State:    model.BindingStatePromoted,
		}},
	}
	if err := os.MkdirAll(filepath.Join(home, "capabilities"), 0o755); err != nil {
		t.Fatalf("create capabilities dir: %v", err)
	}
	if err := jsonfile.WriteAtomic(filepath.Join(home, "capabilities", capability.ID+".json"), &capability); err != nil {
		t.Fatalf("write seeded capability: %v", err)
	}
	return capability
}
