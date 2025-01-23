package cmd

import (
	"fmt"
	"obs-pusher/Patrick-Ivann/observability-pusher/internal/kubernetes"

	"github.com/spf13/cobra"
)

var podLabels Labels

func init() {
	podLabels = Labels{"obs-pusher": "metrics"} // Initialize with default label

	metricsPushCmd.Flags().String("namespace", "test", "Namespace to create resources in")
	metricsPushCmd.Flags().String("element", "", "Name of producing app")
	metricsPushCmd.Flags().String("metric", "", "Name of the metric to push")
	metricsPushCmd.Flags().Int("value", 0, "Value of the metric to push")
	metricsPushCmd.Flags().String("tag-value", "", "")
	metricsPushCmd.Flags().String("tag-label", "", "")
	metricsPushCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)
}

// metricsPushCmd represents the push command for metrics
var metricsPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push a metric",
	Long:  "Push a metric --namespace=<> --element=<> --metric=<> --value=<> --tag=<> --label=<> --pod-labels=key:value,anotherkey:anothervalue",

	Run: func(cmd *cobra.Command, args []string) {
		namespace, _ := cmd.Flags().GetString("namespace")
		elementName, _ := cmd.Flags().GetString("element")
		metricName, _ := cmd.Flags().GetString("metric")
		metricValue, _ := cmd.Flags().GetInt("value")
		metricTagValue, _ := cmd.Flags().GetString("tag-value")
		metricTagLabel, _ := cmd.Flags().GetString("tag-label")

		fmt.Println("Pushing a metric...")

		knImpl, err := kubernetes.NewClientset()
		if err != nil {
			println(err)
			return
		}

		// check if namespace exists
		isNamespaceExisting, err := knImpl.IsNamespaceExisting(namespace)
		if err != nil {
			println(err)
			return
		}
		// create namespace
		if !isNamespaceExisting {
			knImpl.CreateNamespace(namespace)
		}

		// Check if pod exists by fetching it based on labels
		services, err := knImpl.FetchServiceByLabels(namespace, Labels{"obspusher": "metrics"})
		if err != nil {
			println(err)
			return
		}

		// delete existing pod if it exists
		if len(services.Items) > 0 {
			for _, pod := range services.Items {
				knImpl.DeleteService(namespace, pod.Name)
			}
		}

		// Create service with specific name and namespace
		knImpl.CreateService(namespace, elementName, Labels{"obs-pusher": "metrics"})

		// Check if pod exists by fetching it based on labels
		servicemonitors, err := knImpl.FetchServiceMonitorByLabels(namespace, Labels{"obspusher": "metrics"})
		if err != nil {
			println(err)
			return
		}

		// delete existing pod if it exists
		if len(servicemonitors.Items) > 0 {
			for _, pod := range servicemonitors.Items {
				knImpl.DeleteServiceMonitor(namespace, pod.Name)
			}
		}

		knImpl.CreateServiceMonitor(namespace, elementName, Labels{"obs-pusher": "metrics"})

		// Use existing serviceMonitor

		// Check if pod exists by fetching it based on labels
		podList, err := knImpl.FetchPodByLabels(namespace, Labels{"obspusher": "metrics"})
		if err != nil {
			println(err)
			return
		}

		// delete existing pod if it exists
		if len(podList.Items) > 0 {
			for _, pod := range podList.Items {
				knImpl.DeletePod(namespace, pod.Name)
				knImpl.WaitForPodDeletion(namespace, pod.Name)
			}
		}
		// Create pod metric generator and exposing
		knImpl.CreateMetricPod(namespace, elementName, []string{"/bin/sh", "-c", fmt.Sprintf(`while true; do
                                     echo "# HELP %s A custom gauge metric" > /usr/share/nginx/html/metrics;
                                     echo "# TYPE %s gauge" >> /usr/share/nginx/html/metrics;
                                     echo "%s{%s=\"%s\"} %d" >> /usr/share/nginx/html/metrics;
                                     sleep 5;
                                   done`, metricName, metricName, metricName, metricTagLabel, metricTagValue, metricValue)}, podLabels)

	},
}
