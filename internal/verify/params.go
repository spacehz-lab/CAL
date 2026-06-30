package verify

import (
	"fmt"
	"strings"
)

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
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}
