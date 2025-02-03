package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var eventsFilePath string

// eventsCmd represents the events command
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Manage events",
}

func init() {
	podLabels = Labels{"obs-pusher": "events"} // Initialize with default label

	eventsCmd.AddCommand(eventsPushCmd)
	eventsCmd.AddCommand(eventsListCmd)
	eventsCmd.AddCommand(eventsClearCmd)
	eventsCmd.AddCommand(eventsPushSequenceCmd)

	eventsFilePath = *eventsCmd.Flags().String("path", os.Getenv("HOME")+"/.obs-pusher/"+"events.xml", "path of the source xml")
	eventsCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)

}
