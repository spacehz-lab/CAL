package proposalflow

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/config"
	"github.com/spacehz-lab/cal/internal/core"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestLiveLLMStage1OpenSSL(t *testing.T) {
	if os.Getenv("CAL_LIVE_LLM_STAGE1") != "1" {
		t.Skip("set CAL_LIVE_LLM_STAGE1=1 and CAL_LLM_* to run live Stage1 e2e")
	}
	required := []string{config.EnvLLMAPI, config.EnvLLMModel, config.EnvLLMAPIKey}
	for _, name := range required {
		if os.Getenv(name) == "" {
			t.Skipf("set %s to run live Stage1 e2e", name)
		}
	}
	if os.Getenv(config.EnvLLMAPI) != config.LLMAPIChatCompletions {
		t.Skip("live Stage1 e2e requires CAL_LLM_API=chat_completions")
	}

	help, err := exec.Command("openssl", "help").CombinedOutput()
	if err != nil && len(help) == 0 {
		t.Fatalf("openssl help error = %v", err)
	}
	client, err := sharedllm.NewClient(config.LLMFromEnv())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	started := time.Now()
	items, raw, _, err := NewLLMProposer(client).draftSurface(ctx, Request{
		Provider: core.Provider{
			ID:   "provider_openssl",
			Name: "openssl",
			Kind: core.ProviderKindCLI,
			Path: "openssl",
		},
		Observations: []caltrace.Observation{{
			ProviderID: "provider_openssl",
			Type:       "cli_output",
			Source:     "help",
			Content:    map[string]any{"text": string(help)},
		}},
	}, cliProfile())
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("draftSurface() error after %s = %v\nraw=%s", elapsed, err, raw)
	}
	t.Logf("stage1 openssl elapsed=%s raw_bytes=%d kept_surface_items=%d", elapsed, len(raw), len(items))

	surfaces := surfaceNames(items)
	for _, name := range []string{"dgst", "enc", "rand", "x509"} {
		if !surfaces[name] {
			t.Fatalf("surface names = %#v, want %q", surfaces, name)
		}
	}
	for _, name := range []string{"help", "version"} {
		if surfaces[name] {
			t.Fatalf("surface names = %#v, want %q filtered by policy", surfaces, name)
		}
	}
	if len(items) == 0 || len(items) > defaultMaxSurface {
		t.Fatalf("surface count = %d, want 1..%d", len(items), defaultMaxSurface)
	}
}

func surfaceNames(items []surface) map[string]bool {
	names := make(map[string]bool, len(items))
	for _, item := range items {
		names[item.Name] = true
	}
	return names
}
