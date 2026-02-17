package cli

import (
	"github.com/pedramktb/go-gimpl/internal/generator"
	"github.com/spf13/cobra"
)

var GenCmd = &cobra.Command{
	Use:   "gen [path]",
	Short: "Generate gimpl implementations",
	Long:  "Generate gimpl implementations for structs annotated with gimpl tags.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		return generator.Generate(path)
	},
}
