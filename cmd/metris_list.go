package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// metricsListCmd represents the list command for metrics
var metricsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all metrics",
	Run: func(cmd *cobra.Command, args []string) {
		// Add your logic to list metrics here
		fmt.Println("Listing all metrics...")
	},
}
