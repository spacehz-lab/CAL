package proposalflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
)

func (proposer *LLMProposer) planCapabilities(ctx context.Context, req Request, prof profile, surfaces []surfaceItem) ([]capabilityPlanItem, []byte, error) {
	content, err := proposer.client.Complete(ctx, cliCapabilityPrompt(req, prof, surfaces))
	if err != nil {
		return nil, nil, fmt.Errorf("capability stage: %w", err)
	}
	var output capabilityStageOutput
	if err := json.Unmarshal(content, &output); err != nil {
		return nil, content, fmt.Errorf("decode capability stage: %w", err)
	}
	seen := map[string]struct{}{}
	items := make([]capabilityPlanItem, 0, len(output.Capabilities))
	for _, item := range output.Capabilities {
		id := strings.TrimSpace(item.CapabilityID)
		if id == "" || !core.ValidCapabilityID(id) {
			return nil, content, fmt.Errorf("capability stage returned invalid capability_id %q", item.CapabilityID)
		}
		if req.DebugFilter != "" && id != req.DebugFilter {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		item.CapabilityID = id
		seen[id] = struct{}{}
		items = append(items, item)
		if prof.maxCapabilities > 0 && len(items) >= prof.maxCapabilities {
			break
		}
	}
	if len(items) == 0 {
		return nil, content, fmt.Errorf("capability stage returned no capabilities")
	}
	return items, content, nil
}

func existingCapabilityIDs(req Request, limit int) []string {
	ids := make([]string, 0, len(req.Catalog))
	seen := map[string]struct{}{}
	if req.DebugFilter != "" {
		for _, capability := range req.Catalog {
			if capability.ID == req.DebugFilter {
				ids = append(ids, capability.ID)
				seen[capability.ID] = struct{}{}
				break
			}
		}
	}
	for _, capability := range req.Catalog {
		if capability.ID == "" {
			continue
		}
		if _, ok := seen[capability.ID]; ok {
			continue
		}
		ids = append(ids, capability.ID)
		seen[capability.ID] = struct{}{}
		if limit > 0 && len(ids) >= limit {
			break
		}
	}
	return ids
}
