package cli

import (
	"github.com/pedramktb/go-gimpl/internal/generator"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen [path]",
	Short: "Generate gimpl implementations",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		return generator.Generate(path)
	},
}

func init() {
	rootCmd.AddCommand(genCmd)
}
