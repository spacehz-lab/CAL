package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
)

func TestCALCLISmoke(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	pathDir := filepath.Join(temp, "bin")
	if err := os.Mkdir(pathDir, 0o755); err != nil {
		t.Fatalf("create PATH dir: %v", err)
	}
	e2etest.WriteFakeExecutable(t, filepath.Join(pathDir, "soffice"))
	appDir := filepath.Join(temp, "apps")
	if err := os.MkdirAll(filepath.Join(appDir, "Preview.app"), 0o755); err != nil {
		t.Fatalf("create fake app bundle: %v", err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("create CAL_HOME: %v", err)
	}
	e2etest.WriteConfig(t, filepath.Join(home, "config.json"), pathDir, appDir)
	e2etest.WritePDFMagicVerifier(t, home, "file_parse_pdf")

	env := e2etest.WithHomeEnv(os.Environ(), home)

	if output := e2etest.Run(t, repo, env, calctlBin, "--help"); !strings.Contains(output, "Capability Acquisition Loop") {
		t.Fatalf("calctl --help output missing title:\n%s", output)
	}
	if output := e2etest.Run(t, repo, env, caldBin, "--help"); !strings.Contains(output, "CAL local service") {
		t.Fatalf("cald --help output missing title:\n%s", output)
	}
	if output := e2etest.Run(t, repo, env, caldBin, "serve", "--help"); !strings.Contains(output, "Start the local CAL service") {
		t.Fatalf("cald serve --help output missing title:\n%s", output)
	}

	var status struct {
		Running bool   `json:"running"`
		Mode    string `json:"mode"`
	}
	e2etest.RunJSON(t, repo, env, &status, calctlBin, "daemon", "status", "--json")
	if status.Running || status.Mode != "local" {
		t.Fatalf("daemon status = %#v, want stopped local status", status)
	}

	e2etest.StartCald(t, repo, env, caldBin)

	e2etest.RunJSON(t, repo, env, &status, calctlBin, "daemon", "status", "--json")
	if !status.Running || status.Mode != "local" {
		t.Fatalf("daemon status = %#v, want running local status", status)
	}

	var sources struct {
		Sources []struct {
			Kind  string `json:"kind"`
			Value string `json:"value"`
		} `json:"sources"`
	}
	e2etest.RunJSON(t, repo, env, &sources, calctlBin, "providers", "sources", "list", "--json")
	if len(sources.Sources) == 0 {
		t.Fatalf("provider sources = %#v, want configured sources", sources)
	}

	var cliFind struct {
		ProvidersCreated int                       `json:"providers_created"`
		Providers        []e2etest.ProviderSummary `json:"providers"`
	}
	e2etest.RunJSON(t, repo, env, &cliFind, calctlBin, "providers", "find", "--kind", "cli", "--json")
	if cliFind.ProvidersCreated != 1 || len(cliFind.Providers) != 1 {
		t.Fatalf("cli find = %#v, want one created cli provider", cliFind)
	}
	soffice, ok := e2etest.FindProvider(cliFind.Providers, "soffice", "cli")
	if !ok {
		t.Fatalf("cli find = %#v, want soffice cli provider", cliFind)
	}

	var appFind struct {
		ProvidersCreated int                       `json:"providers_created"`
		Providers        []e2etest.ProviderSummary `json:"providers"`
	}
	e2etest.RunJSON(t, repo, env, &appFind, calctlBin, "providers", "find", "--kind", "app", "--json")
	if appFind.ProvidersCreated != 1 || len(appFind.Providers) != 1 {
		t.Fatalf("app find = %#v, want one created app provider", appFind)
	}
	preview, ok := e2etest.FindProvider(appFind.Providers, "Preview", "app")
	if !ok {
		t.Fatalf("app find = %#v, want Preview app provider from configured app dir", appFind)
	}

	var discovery struct {
		JobID                string                    `json:"job_id"`
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		ProvidersCreated     int                       `json:"providers_created"`
		ProvidersUpdated     int                       `json:"providers_updated"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	e2etest.RunJSON(t, repo, env, &discovery, calctlBin, "discovery", "run", "--provider-id", soffice.ID, "--mode", "rules", "--json")
	if !strings.HasPrefix(discovery.JobID, "disc_") || !strings.HasPrefix(discovery.TraceID, "trace_") || discovery.State != "succeeded" || discovery.CapabilitiesPromoted != 1 || discovery.BindingsPromoted != 1 {
		t.Fatalf("discover = %#v, want succeeded acquisition job with created providers and promoted binding", discovery)
	}
	if _, err := os.Stat(filepath.Join(home, "discovery", discovery.TraceID, "trace.json")); err != nil {
		t.Fatalf("trace file missing: %v", err)
	}
	if !strings.HasPrefix(soffice.ID, "provider_") || soffice.Path != filepath.Join(pathDir, "soffice") {
		t.Fatalf("soffice provider = %#v, want entry-scoped provider id and path", soffice)
	}
	if !strings.HasPrefix(preview.ID, "provider_") || preview.Path != filepath.Join(appDir, "Preview.app") {
		t.Fatalf("preview provider = %#v, want entry-scoped provider id and path", preview)
	}

	var storedProviders struct {
		Providers []e2etest.ProviderSummary `json:"providers"`
	}
	e2etest.RunJSON(t, repo, env, &storedProviders, calctlBin, "providers", "list", "--json")
	if len(storedProviders.Providers) != 2 {
		t.Fatalf("providers list = %#v, want two stored providers", storedProviders)
	}
	var providerDetail e2etest.ProviderSummary
	e2etest.RunJSON(t, repo, env, &providerDetail, calctlBin, "providers", "get", "--provider-id", soffice.ID, "--json")
	if providerDetail.ID != soffice.ID || providerDetail.Path != soffice.Path {
		t.Fatalf("provider detail = %#v, want soffice provider", providerDetail)
	}

	var capabilities struct {
		Count        int   `json:"count"`
		Capabilities []any `json:"capabilities"`
	}
	e2etest.RunJSON(t, repo, env, &capabilities, calctlBin, "capabilities", "list", "--json")
	if capabilities.Count != 1 || len(capabilities.Capabilities) != 1 {
		t.Fatalf("capability list = %#v, want one promoted capability", capabilities)
	}
	e2etest.RunJSON(t, repo, env, &capabilities, calctlBin, "capabilities", "list", "--provider-id", soffice.ID, "--json")
	if capabilities.Count != 1 || len(capabilities.Capabilities) != 1 {
		t.Fatalf("capability list by provider = %#v, want one promoted capability", capabilities)
	}
	var capabilityDetail struct {
		ID string `json:"id"`
	}
	e2etest.RunJSON(t, repo, env, &capabilityDetail, calctlBin, "capabilities", "get", "--capability-id", "document.export_pdf", "--json")
	if capabilityDetail.ID != "document.export_pdf" {
		t.Fatalf("capability detail = %#v, want document.export_pdf", capabilityDetail)
	}

	var eval struct {
		Summary struct {
			Providers        int `json:"providers"`
			Capabilities     int `json:"capabilities"`
			Bindings         int `json:"bindings"`
			PromotedBindings int `json:"promoted_bindings"`
			Traces           int `json:"traces"`
			Runs             int `json:"runs"`
		} `json:"summary"`
	}
	e2etest.RunJSON(t, repo, env, &eval, calctlBin, "eval", "--json")
	if eval.Summary.Providers != 2 || eval.Summary.Capabilities != 1 || eval.Summary.Bindings != 1 || eval.Summary.PromotedBindings != 1 || eval.Summary.Traces != 1 || eval.Summary.Runs != 0 {
		t.Fatalf("eval = %#v, want acquisition counts after scan", eval)
	}

	var acquisition struct {
		JobID                string `json:"job_id"`
		State                string `json:"state"`
		TraceID              string `json:"trace_id"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	e2etest.RunJSON(t, repo, env, &acquisition, calctlBin, "discovery", "run", "--provider-id", soffice.ID, "--mode", "rules", "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 0 || acquisition.BindingsPromoted != 1 || !strings.HasPrefix(acquisition.TraceID, "trace_") {
		t.Fatalf("acquisition discovery = %#v, want refreshed binding for existing capability", acquisition)
	}
	if _, err := os.Stat(filepath.Join(home, "discovery", acquisition.TraceID, "trace.json")); err != nil {
		t.Fatalf("acquisition trace file missing: %v", err)
	}

	e2etest.RunJSON(t, repo, env, &capabilities, calctlBin, "capabilities", "list", "--json")
	if capabilities.Count != 1 || len(capabilities.Capabilities) != 1 {
		t.Fatalf("capability list = %#v, want one promoted capability", capabilities)
	}

	source := filepath.Join(temp, "source.txt")
	target := filepath.Join(temp, "target.pdf")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write run source: %v", err)
	}
	var runSuccess struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		Verified     bool   `json:"verified"`
		CapabilityID string `json:"capability_id"`
		BindingID    string `json:"binding_id"`
		ProviderID   string `json:"provider_id"`
	}
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "document.export_pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified || runSuccess.CapabilityID != "document.export_pdf" || runSuccess.BindingID == "" || runSuccess.ProviderID != soffice.ID {
		t.Fatalf("run success = %#v, want verified successful reuse", runSuccess)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("run target missing: %v", err)
	}

	var useSuccess struct {
		Status    string `json:"status"`
		Selection struct {
			CapabilityID string `json:"capability_id"`
			BindingID    string `json:"binding_id"`
			ProviderID   string `json:"provider_id"`
		} `json:"selection"`
		Run struct {
			ID           string         `json:"id"`
			Status       string         `json:"status"`
			Verified     bool           `json:"verified"`
			CapabilityID string         `json:"capability_id"`
			BindingID    string         `json:"binding_id"`
			ProviderID   string         `json:"provider_id"`
			Inputs       map[string]any `json:"inputs"`
		} `json:"run"`
	}
	e2etest.RunJSON(t, repo, env, &useSuccess, calctlBin, "use", "--intent", "export this document as pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`}`, "--verify", "--json")
	if useSuccess.Status != "succeeded" || useSuccess.Selection.CapabilityID != "document.export_pdf" || useSuccess.Selection.BindingID == "" || useSuccess.Selection.ProviderID != soffice.ID {
		t.Fatalf("use success = %#v, want selected document.export_pdf binding", useSuccess)
	}
	if useSuccess.Run.Status != "succeeded" || !useSuccess.Run.Verified || useSuccess.Run.BindingID != useSuccess.Selection.BindingID || useSuccess.Run.ProviderID != soffice.ID {
		t.Fatalf("use run = %#v, want verified selected binding run", useSuccess.Run)
	}
	useTarget, ok := useSuccess.Run.Inputs["target"].(string)
	if !ok || useTarget == "" {
		t.Fatalf("use run inputs = %#v, want generated target", useSuccess.Run.Inputs)
	}
	if _, err := os.Stat(useTarget); err != nil {
		t.Fatalf("use target missing: %v", err)
	}

	type runDetailOutput struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	var runDetail runDetailOutput
	e2etest.RunJSON(t, repo, env, &runDetail, calctlBin, "runs", "get", "--run-id", runSuccess.ID, "--json")
	if runDetail.ID != runSuccess.ID || runDetail.Status != "succeeded" {
		t.Fatalf("run detail = %#v, want stored run %s", runDetail, runSuccess.ID)
	}
	var traceDetail struct {
		ID string `json:"id"`
	}
	e2etest.RunJSON(t, repo, env, &traceDetail, calctlBin, "traces", "get", "--trace-id", acquisition.TraceID, "--json")
	if traceDetail.ID != acquisition.TraceID {
		t.Fatalf("trace detail = %#v, want trace %s", traceDetail, acquisition.TraceID)
	}

	e2etest.RunJSON(t, repo, env, &eval, calctlBin, "eval", "--json")
	if eval.Summary.Providers != 2 || eval.Summary.Capabilities != 1 || eval.Summary.Bindings != 1 || eval.Summary.PromotedBindings != 1 || eval.Summary.Traces != 2 || eval.Summary.Runs != 2 {
		t.Fatalf("eval = %#v, want closed-loop counts", eval)
	}

	var daemonFailure struct {
		Running bool `json:"running"`
	}
	e2etest.RunJSON(t, repo, env, &daemonFailure, calctlBin, "daemon", "start", "--json")
	if !daemonFailure.Running {
		t.Fatalf("daemon start = %#v, want idempotent running status", daemonFailure)
	}
}
