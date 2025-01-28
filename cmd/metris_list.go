package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Patrick-Ivann/observability-pusher/internal/sources"

	"github.com/spf13/cobra"
)

// metricsListCmd represents the list command for metrics
var metricsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all metrics",
	Run: func(cmd *cobra.Command, args []string) {

		dictionary, err := sources.ReadDictionary(metricFilePath)
		if err != nil {
			println(err.Error())
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "ID\tDESCRIPTION\tTAGS\tTYPE")
		for _, metric := range dictionary.Metrics {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				metric.Name, metric.Description, metric.Tags, metric.Type)
		}
		w.Flush()

	},
}
