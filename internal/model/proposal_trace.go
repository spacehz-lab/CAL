package model

// ProposalTrace records proposal-stage diagnostics that are not executable candidates.
type ProposalTrace struct {
	SchemaVersion string            `json:"schema_version,omitempty"`
	PromptVersion string            `json:"prompt_version,omitempty"`
	Model         string            `json:"model,omitempty"`
	Stages        []ProposalStage   `json:"stages,omitempty"`
	Attempts      []ProposalAttempt `json:"attempts,omitempty"`
}

// ProposalStageName identifies a proposal diagnostics stage.
type ProposalStageName string

const (
	// ProposalStageSurface records surface inventory decisions.
	ProposalStageSurface ProposalStageName = "surface"
	// ProposalStageCapability records capability planning decisions.
	ProposalStageCapability ProposalStageName = "capability"
	// ProposalStageBinding records candidate binding decisions.
	ProposalStageBinding ProposalStageName = "binding"
	// ProposalStageEvidence identifies verification planning work.
	ProposalStageEvidence ProposalStageName = "evidence"
)

// ProposalSummaryKey identifies proposal-stage summary counters.
type ProposalSummaryKey string

const (
	// ProposalSummaryRaw counts parsed items before local filtering.
	ProposalSummaryRaw ProposalSummaryKey = "raw"
	// ProposalSummaryKeep counts items whose final decision is keep.
	ProposalSummaryKeep ProposalSummaryKey = "keep"
	// ProposalSummaryDefer counts items whose final decision is defer.
	ProposalSummaryDefer ProposalSummaryKey = "defer"
	// ProposalSummarySkip counts items whose final decision is skip.
	ProposalSummarySkip ProposalSummaryKey = "skip"
	// ProposalSummarySelected counts items passed to the next Proposal stage.
	ProposalSummarySelected ProposalSummaryKey = "selected"
	// ProposalSummaryReused counts items that reuse an existing catalog capability id.
	ProposalSummaryReused ProposalSummaryKey = "reused"
	// ProposalSummaryCreated counts items that introduce a new capability id.
	ProposalSummaryCreated ProposalSummaryKey = "created"
	// ProposalSummaryOutOfPolicy counts kept items using a subject or operation outside the policy range.
	ProposalSummaryOutOfPolicy ProposalSummaryKey = "out_of_policy"
)

// ProposalStage records one proposal stage's parsed decisions.
type ProposalStage struct {
	Name       ProposalStageName          `json:"name"`
	Items      []ProposalItem             `json:"items,omitempty"`
	Summary    map[ProposalSummaryKey]int `json:"summary,omitempty"`
	DurationMS int64                      `json:"duration_ms,omitempty"`
}

// ProposalAttemptStatus identifies whether one Proposal LLM call produced usable output.
type ProposalAttemptStatus string

const (
	// ProposalAttemptSucceeded marks one successful Proposal LLM call.
	ProposalAttemptSucceeded ProposalAttemptStatus = "succeeded"
	// ProposalAttemptFailed marks one failed Proposal LLM call.
	ProposalAttemptFailed ProposalAttemptStatus = "failed"
)

// ProposalAttempt records raw LLM response diagnostics for one Proposal stage call.
type ProposalAttempt struct {
	Stage            ProposalStageName     `json:"stage"`
	CapabilityID     string                `json:"capability_id,omitempty"`
	CandidateIndex   *int                  `json:"candidate_index,omitempty"`
	Status           ProposalAttemptStatus `json:"status"`
	DurationMS       int64                 `json:"duration_ms,omitempty"`
	PromptTokens     int64                 `json:"prompt_tokens,omitempty"`
	CompletionTokens int64                 `json:"completion_tokens,omitempty"`
	TotalTokens      int64                 `json:"total_tokens,omitempty"`
	Error            *RecordError          `json:"error,omitempty"`
	RawResponse      string                `json:"raw_response,omitempty"`
}

// ProposalDecision records whether a proposal item should continue to the next stage.
type ProposalDecision string

const (
	// ProposalDecisionKeep marks an item selected for the next Proposal stage.
	ProposalDecisionKeep ProposalDecision = "keep"
	// ProposalDecisionDefer marks an item observed but deferred from this run.
	ProposalDecisionDefer ProposalDecision = "defer"
	// ProposalDecisionSkip marks an item filtered out of Proposal planning.
	ProposalDecisionSkip ProposalDecision = "skip"
)

// ProposalItem records a non-executable proposal-stage decision.
type ProposalItem struct {
	ID       string           `json:"id,omitempty"`
	Kind     string           `json:"kind,omitempty"`
	Name     string           `json:"name,omitempty"`
	Decision ProposalDecision `json:"decision,omitempty"`
	Reason   string           `json:"reason,omitempty"`
}
