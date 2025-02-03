package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var podLabels Labels
var metricFilePath string

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Manage metrics",
}

func init() {
	podLabels = Labels{"obs-pusher": "metrics"} // Initialize with default label
	metricFilePath = *metricsCmd.Flags().String("path", os.Getenv("HOME")+"/.obs-pusher/"+"metrics.xml", "path of the source xml")

	metricsCmd.AddCommand(metricsListCmd)
	metricsCmd.AddCommand(metricsPushCmd)
	metricsCmd.AddCommand(metricsClearCmd)
	metricsCmd.AddCommand(metricsPushDictionaryCmd)
	metricsCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)

}
