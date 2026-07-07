package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func parseInputsJSON(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("parse inputs json: %w", err)
	}
	inputs, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("inputs json must be an object")
	}
	return inputs, nil
}

func parseMinVerifyLevel(raw string) (model.VerifyLevel, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	level := model.VerifyLevel(raw)
	switch level {
	case model.VerifyLevelL0, model.VerifyLevelL1, model.VerifyLevelL2, model.VerifyLevelL3:
		return level, nil
	default:
		return "", fmt.Errorf("min verify level must be one of L0, L1, L2, L3")
	}
}
