package evidence

import (
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	paramValue     = "value"
	paramValues    = "values"
	paramPattern   = "pattern"
	paramFormat    = "format"
	paramSource    = "source"
	paramQuery     = "query"
	paramTransform = "transform"
	paramAlgorithm = "algorithm"
)

const (
	formatPDF  = "pdf"
	formatPNG  = "png"
	formatJSON = "json"
	formatText = "text"
	formatZIP  = "zip"
)

const (
	transformBase64Encode = "base64_encode"
	transformBase64Decode = "base64_decode"
)

const (
	hashSHA1        = "sha1"
	hashSHA256      = "sha256"
	hashSHA1Dash    = "sha-1"
	hashSHA256Dash  = "sha-256"
	hashSHA1Under   = "sha_1"
	hashSHA256Under = "sha_256"
	hashSHA1Space   = "sha 1"
	hashSHA256Space = "sha 256"
)

// PredicateRule is the prompt-facing and parser-facing parameter contract for one predicate.
type PredicateRule struct {
	Predicate      model.VerifyPredicate `json:"predicate"`
	RequiredParams []string              `json:"required_params,omitempty"`
	AllowedParams  []string              `json:"allowed_params,omitempty"`
	AllowedValues  map[string][]string   `json:"allowed_values,omitempty"`
}

func verifyPredicateRules() []PredicateRule {
	return []PredicateRule{
		{Predicate: model.VerifyPredicateEquals, RequiredParams: []string{paramValue}, AllowedParams: []string{paramValue}},
		{Predicate: model.VerifyPredicateNotEquals, RequiredParams: []string{paramValue}, AllowedParams: []string{paramValue}},
		{Predicate: model.VerifyPredicateExists},
		{Predicate: model.VerifyPredicateNonEmpty},
		{
			Predicate:      model.VerifyPredicateFormat,
			RequiredParams: []string{paramFormat},
			AllowedParams:  []string{paramFormat},
			AllowedValues:  map[string][]string{paramFormat: []string{formatPDF, formatPNG, formatJSON, formatText}},
		},
		{Predicate: model.VerifyPredicateContains, RequiredParams: []string{paramValue}, AllowedParams: []string{paramValue}},
		{Predicate: model.VerifyPredicateContainsAny, RequiredParams: []string{paramValues}, AllowedParams: []string{paramValues}},
		{Predicate: model.VerifyPredicateRegex, RequiredParams: []string{paramPattern}, AllowedParams: []string{paramPattern}},
		{
			Predicate:      model.VerifyPredicateBytesEqualTransform,
			RequiredParams: []string{paramSource, paramTransform},
			AllowedParams:  []string{paramSource, paramTransform},
			AllowedValues:  map[string][]string{paramTransform: []string{transformBase64Encode, transformBase64Decode}},
		},
		{
			Predicate:      model.VerifyPredicateHashLineMatches,
			RequiredParams: []string{paramSource, paramAlgorithm},
			AllowedParams:  []string{paramSource, paramAlgorithm},
			AllowedValues:  map[string][]string{paramAlgorithm: []string{hashSHA1, hashSHA256, hashSHA1Dash, hashSHA256Dash, hashSHA1Under, hashSHA256Under, hashSHA1Space, hashSHA256Space}},
		},
		{
			Predicate:      model.VerifyPredicateArchiveContainsInput,
			RequiredParams: []string{paramSource, paramFormat},
			AllowedParams:  []string{paramSource, paramFormat},
			AllowedValues:  map[string][]string{paramFormat: []string{formatZIP}},
		},
		{
			Predicate:      model.VerifyPredicateJSONQueryMatches,
			RequiredParams: []string{paramSource, paramQuery},
			AllowedParams:  []string{paramSource, paramQuery},
		},
	}
}

func validatePredicateParams(check model.VerifyCheck, rules map[model.VerifyPredicate]PredicateRule) string {
	rule, ok := rules[check.Predicate]
	if !ok {
		return "unknown predicate"
	}
	params := check.Params
	for _, name := range rule.RequiredParams {
		if params == nil || params[name] == nil || fmt.Sprint(params[name]) == "" {
			return "missing predicate param"
		}
	}
	allowed := stringSet(rule.AllowedParams)
	for name, value := range params {
		if !allowed[name] {
			return "unknown predicate param"
		}
		if values := rule.AllowedValues[name]; len(values) > 0 && !valueAllowed(value, values) {
			return "invalid predicate param"
		}
	}
	return ""
}

func predicateRuleMap(rules []PredicateRule) map[model.VerifyPredicate]PredicateRule {
	result := make(map[model.VerifyPredicate]PredicateRule, len(rules))
	for _, rule := range rules {
		if rule.Predicate != "" {
			result[rule.Predicate] = rule
		}
	}
	return result
}

func stringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		if value != "" {
			result[value] = true
		}
	}
	return result
}

func valueAllowed(value any, allowed []string) bool {
	text := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
	for _, item := range allowed {
		if text == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}
