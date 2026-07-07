package check

import (
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func validateParams(check *model.VerifyCheck, rules []paramRule) error {
	for _, rule := range rules {
		if rule.required {
			switch rule.name {
			case paramValues:
				if len(stringListParam(check.Params, rule.name)) == 0 {
					return fmt.Errorf("verify predicate %s requires params.%s", check.Predicate, rule.name)
				}
			default:
				if _, ok := check.Params[rule.name]; !ok {
					return fmt.Errorf("verify predicate %s requires params.%s", check.Predicate, rule.name)
				}
			}
		}
		if len(rule.allowedValues) > 0 {
			value := stringParam(check.Params, rule.name)
			if value == "" || !paramValueAllowed(value, rule.allowedValues) {
				return fmt.Errorf("verify predicate %s params.%s %q is not supported", check.Predicate, rule.name, value)
			}
		}
	}
	return nil
}

func stringParam(params map[string]any, key string) string {
	if len(params) == 0 {
		return ""
	}
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func pathParam(params map[string]any, key string, inputs map[string]any) string {
	value := stringParam(params, key)
	if inputPath, ok := inputs[value].(string); ok && strings.TrimSpace(inputPath) != "" {
		return strings.TrimSpace(inputPath)
	}
	return value
}

func stringListParam(params map[string]any, key string) []string {
	value, ok := params[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		values := make([]string, 0, len(typed))
		for _, value := range typed {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				values = append(values, trimmed)
			}
		}
		return values
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func paramValueAllowed(value string, allowedValues []string) bool {
	for _, allowed := range allowedValues {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}
