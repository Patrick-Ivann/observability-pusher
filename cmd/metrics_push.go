package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// metricsPushCmd represents the push command for metrics
var metricsPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push a metric",
	Run: func(cmd *cobra.Command, args []string) {
		// Add your logic to push a metric here
		fmt.Println("Pushing a metric...")
	},
}
