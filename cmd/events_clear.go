package cmd

import (
	"github.com/Patrick-Ivann/observability-pusher/internal/kubernetes"
	"github.com/spf13/cobra"
)

func init() {
	eventsClearCmd.Flags().String("namespace", "", "Namespace to fetch resources in, if no value will scan the whole cluster")
	eventsClearCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)

}

// eventsClearCmd represents the clear command for events
var eventsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear events related objects",
	Long:  "Clear events related objects",
	Run: func(cmd *cobra.Command, args []string) {
		namespace, _ := cmd.Flags().GetString("namespace")

		podLabels.Append(Labels{"obs-pusher": "events"})
		knImpl, err := kubernetes.NewClientset("", "", "")
		if err != nil {
			println(err.Error())
			return
		}

		// Check if pod exists by fetching it based on labels
		podList, err := knImpl.FetchPodByLabels(namespace, podLabels)
		if err != nil {
			println(err.Error())
			return
		}

		// delete existing pod if it exists
		if len(podList.Items) > 0 {
			for _, pod := range podList.Items {
				knImpl.DeletePod(pod.Namespace, pod.Name)
				knImpl.WaitForPodDeletion(pod.Namespace, pod.Name)
			}
		}
	},
}
