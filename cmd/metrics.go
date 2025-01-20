package cmd

import (
	"github.com/spf13/cobra"
)

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Manage metrics",
}

func init() {
	metricsCmd.AddCommand(metricsListCmd)
	metricsCmd.AddCommand(metricsPushCmd)
}
