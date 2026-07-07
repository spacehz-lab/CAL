package check

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func registerBytesPredicates(c *Checker) {
	c.register(predicate{
		name:     model.VerifyPredicateBytesEqualTransform,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile},
		params: []paramRule{
			{name: paramSource, required: true},
			{name: paramTransform, required: true, allowedValues: []string{transformBase64Encode, transformBase64Decode}},
		},
		run: checkBytesEqualTransform,
	})
}

func checkBytesEqualTransform(ctx *predicateContext) error {
	source := pathParam(ctx.check.Params, paramSource, ctx.subject.inputs)
	transform := stringParam(ctx.check.Params, paramTransform)
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	targetBytes, err := os.ReadFile(ctx.subject.path)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	expected, err := transformBytes(sourceBytes, transform)
	if err != nil {
		return err
	}
	if !bytes.Equal(bytes.TrimSpace(targetBytes), bytes.TrimSpace(expected)) {
		return fmt.Errorf("verify bytes_equal_transform failed")
	}
	return nil
}

func transformBytes(content []byte, transform string) ([]byte, error) {
	switch strings.ToLower(transform) {
	case transformBase64Encode:
		out := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
		base64.StdEncoding.Encode(out, content)
		return out, nil
	case transformBase64Decode:
		out := make([]byte, base64.StdEncoding.DecodedLen(len(content)))
		n, err := base64.StdEncoding.Decode(out, bytes.TrimSpace(content))
		if err != nil {
			return nil, err
		}
		return out[:n], nil
	default:
		return nil, fmt.Errorf("verify transform %q is not supported", transform)
	}
}
