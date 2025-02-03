package cmd

import (
	"github.com/Patrick-Ivann/observability-pusher/internal/kubernetes"
	"github.com/spf13/cobra"
)

func int() {
	metricsClearCmd.Flags().String("namespace", "", "Namespace to fetch resources in, if no value will scan the whole cluster")
	metricsClearCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)
}

var metricsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear metrics related objects",
	Long:  "Clear metrics related objects",
	Run: func(cmd *cobra.Command, args []string) {

		namespace, _ := cmd.Flags().GetString("namespace")

		podLabels.Append(Labels{"obs-pusher": "metrics"})
		knImpl, err := kubernetes.NewClientset("", "", "")
		if err != nil {
			println(err.Error())
			return
		}

		// Check if pod exists by fetching it based on labels
		services, err := knImpl.FetchServiceByLabels(namespace, podLabels)
		if err != nil {
			println(err.Error())
			return
		}

		// delete existing pod if it exists
		if len(services.Items) > 0 {
			for _, service := range services.Items {
				knImpl.DeleteService(service.Namespace, service.Name)
			}
		}

		// Check if pod exists by fetching it based on labels
		servicemonitors, err := knImpl.FetchServiceMonitorByLabels(namespace, podLabels)
		if err != nil {
			println(err.Error())
			return
		}

		// delete existing pod if it exists
		if len(servicemonitors.Items) > 0 {
			for _, serviceMonitor := range servicemonitors.Items {
				knImpl.DeleteServiceMonitor(serviceMonitor.Namespace, serviceMonitor.Name)
			}
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
