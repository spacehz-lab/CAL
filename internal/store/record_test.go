package store

import "testing"

func TestValidateRecordID(t *testing.T) {
	for _, id := range []string{"", " ", ".", "..", "../bad", `bad\id`} {
		if err := validateRecordID(id); err == nil {
			t.Fatalf("validateRecordID(%q) error = nil, want error", id)
		}
	}
	if err := validateRecordID("record_abc123"); err != nil {
		t.Fatalf("validateRecordID(valid) error = %v", err)
	}
}
