package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Patrick-Ivann/observability-pusher/internal/sources"

	"github.com/spf13/cobra"
)

func init() {
	eventsListCmd.Flags().StringVarP(&eventFilePath, "event-file", "f", os.Getenv("HOME")+"/.obs-pusher/"+"events.xml", "Path to the XML file for the event")
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Long:  "List events from dictionary",
	Run: func(cmd *cobra.Command, args []string) {
		dictionary, err := sources.ReadDictionary(eventFilePath)
		if err != nil {
			println(err.Error())
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "ID\tTEXT")
		for _, notification := range dictionary.Logs {
			fmt.Fprintf(w, "%s\t%s\n",
				notification.ID, notification.Text)
		}
		w.Flush()

	},
}
