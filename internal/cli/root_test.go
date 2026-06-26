package cli

import (
	"bytes"
	"io"
)

func executeRoot(home string, args ...string) (string, error) {
	var out bytes.Buffer
	cmd := NewRootCommand(Config{Home: home, Out: &out, Err: io.Discard})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
