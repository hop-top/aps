package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"oss-aps-cli/internal/core"

	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation",
	Run: func(cmd *cobra.Command, args []string) {
		agentsDir, err := core.GetAgentsDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting agents dir: %v\n", err)
			os.Exit(1)
		}

		docsDest := filepath.Join(agentsDir, "docs")
		if err := core.GenerateDocs(docsDest); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating docs: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Documentation generated at: %s\n", docsDest)
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
