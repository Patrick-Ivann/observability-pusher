package cmd

import (
	"github.com/spf13/cobra"
)

// eventsCmd represents the metrics command
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Manage events",
}

func init() {
	eventsCmd.AddCommand(eventsPushCmd)
	eventsCmd.AddCommand(eventsListCmd)
}
