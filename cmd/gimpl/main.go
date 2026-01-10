package main

import (
	"os"
	"runtime/debug"

	"github.com/pedramktb/go-gimpl/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}

	cmd := &cobra.Command{
		Use:     "gimpl [command]",
		Short:   "gimpl is a code generation tool for go-gimpl.",
		Long:    "gimpl is a code generation tool for go-gimpl.",
		Version: version,
	}

	cmd.AddCommand(cli.GenCmd)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
