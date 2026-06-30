package eval

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestRecordsSummaryCountsDurableRecords(t *testing.T) {
	metrics := records{
		providers: []core.Provider{
			{ID: "provider_one", Kind: core.ProviderKindCLI, Path: "/tmp/one"},
			{ID: "provider_two", Kind: core.ProviderKindCLI, Path: "/tmp/two"},
		},
		capabilities: []core.Capability{
			{
				ID: "document.convert",
				Bindings: []core.Binding{
					{ID: "binding_promoted", State: core.BindingStatePromoted},
					{ID: "binding_unpromoted"},
				},
			},
		},
		runs:   []core.Run{{ID: "run_one"}},
		traces: []caltrace.Trace{{ID: "trace_one"}},
	}.summary()

	if metrics.Providers != 2 || metrics.Capabilities != 1 || metrics.Bindings != 2 || metrics.PromotedBindings != 1 || metrics.Runs != 1 || metrics.Traces != 1 {
		t.Fatalf("summary = %#v, want durable record counts", metrics)
	}
}
