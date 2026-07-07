package contract

import "github.com/spacehz-lab/cal/internal/model"

// AcquisitionMode selects the acquisition proposal source exposed over transport.
type AcquisitionMode string

const (
	AcquisitionModeLive   AcquisitionMode = "live"
	AcquisitionModeReplay AcquisitionMode = "replay"
	AcquisitionModeRules  AcquisitionMode = "rules"
)

// AcquisitionRequest starts one acquisition run.
type AcquisitionRequest struct {
	ProviderID   string          `json:"provider_id,omitempty"`
	Hint         string          `json:"hint,omitempty"`
	ProposalPath string          `json:"proposal_path,omitempty"`
	Mode         AcquisitionMode `json:"mode,omitempty"`
}

// AcquisitionResponse reports acquisition trace and promotion summary.
type AcquisitionResponse struct {
	TraceID              string             `json:"trace_id"`
	ProviderIDs          []string           `json:"provider_ids,omitempty"`
	CapabilitiesPromoted int                `json:"capabilities_promoted"`
	BindingsPromoted     int                `json:"bindings_promoted"`
	Trace                *model.Trace       `json:"trace,omitempty"`
	Error                *model.RecordError `json:"error,omitempty"`
}
