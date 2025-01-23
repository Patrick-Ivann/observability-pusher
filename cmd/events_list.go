package cmd

import (
	"fmt"
	"obs-pusher/Patrick-Ivann/observability-pusher/internal/sources"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	eventsListCmd.Flags().String("path", os.Getenv("HOME"), "path of the source xml")

}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Long:  "List events from dictionary",
	Run: func(cmd *cobra.Command, args []string) {
		filePath, _ := cmd.Flags().GetString("path")

		notifications, err := sources.ReadEventsList(filePath + "/.obs-pusher/" + "dictionary.xml")
		if err != nil {
			println(err.Error())
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
		fmt.Fprintln(w, "ID\tTEXT")
		for _, notification := range notifications {
			fmt.Fprintf(w, "%s\t%s\n",
				notification.ID, notification.Text)
		}
		w.Flush()

	},
}
