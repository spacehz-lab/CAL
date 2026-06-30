package verify

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
)

func checkBytesEqualTransform(check core.VerifyCheck, subject checkSubject) error {
	source := pathParam(check.Params, "source", subject.inputs)
	transform := stringParam(check.Params, "transform")
	if source == "" || transform == "" {
		return fmt.Errorf("verify bytes_equal_transform requires source and transform")
	}
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	targetBytes, err := os.ReadFile(subject.path)
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
	case "base64_encode":
		out := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
		base64.StdEncoding.Encode(out, content)
		return out, nil
	case "base64_decode":
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
