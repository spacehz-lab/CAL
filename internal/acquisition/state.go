package acquisition

import (
	"fmt"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/tracelog"
)

type state struct {
	TraceID      string
	StartedAt    string
	Provider     *model.Provider
	Catalog      []model.Capability
	Observations []model.Observation
	Proposal     *model.ProposalTrace
	Candidates   []model.Candidate
	ProbePlans   []proposal.ProbePlan
	Probes       []model.Probe
	Promotions   []model.Promotion
}

func (st *state) traceRequest(req *Request, recordErr *model.RecordError) *tracelog.Request {
	return &tracelog.Request{
		TraceID:      st.TraceID,
		StartedAt:    st.StartedAt,
		Hint:         req.Hint,
		ProviderIDs:  st.providerIDs(),
		Observations: st.Observations,
		Proposal:     st.Proposal,
		Candidates:   st.Candidates,
		Probes:       st.Probes,
		Promotions:   st.Promotions,
		Error:        recordErr,
	}
}

func (st *state) providerIDs() []string {
	if st.Provider == nil || st.Provider.ID == "" {
		return nil
	}
	return []string{st.Provider.ID}
}

func (st *state) providerID(req *Request) string {
	if st != nil && st.Provider != nil && st.Provider.ID != "" {
		return st.Provider.ID
	}
	if req == nil {
		return ""
	}
	return req.ProviderID
}

func (st *state) traceID() string {
	if st == nil {
		return ""
	}
	return st.TraceID
}

func recordError(code string, message string, err error) *model.RecordError {
	if err == nil {
		return &model.RecordError{Code: code, Message: message}
	}
	return &model.RecordError{Code: code, Message: fmt.Sprintf("%s: %v", message, err)}
}
