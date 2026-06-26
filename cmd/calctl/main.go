package main

import (
	"fmt"
	"os"

	"github.com/spacehz-lab/cal/internal/cli"
)

func main() {
	cmd := cli.NewRootCommand(cli.Config{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	})
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
