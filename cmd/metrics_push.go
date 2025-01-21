package cmd

import (
	"fmt"
	"obs-pusher/internal/kubernetes"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
)

var podLabel Labels

func init() {
	metricsPushCmd.Flags().String("namespace", "test", "Namespace to create resources in")
	metricsPushCmd.Flags().String("element", "", "Name of producing app")
	metricsPushCmd.Flags().String("metric", "", "Name of the metric to push")
	metricsPushCmd.Flags().String("value", "", "Value of the metric to push")
	metricsPushCmd.Flags().String("label", "", "Helping label of the metric to push")
	metricsPushCmd.Flags().String("tag", "", "Helping label of the metric to push")
	metricsPushCmd.Flags().Var(&podLabel, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)
}

// metricsPushCmd represents the push command for metrics
var metricsPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push a metric --namespace=<> --element=<> --metric=<> --value=<> --tag=<> --label=<> --pod-labels=key:value,anotherkey:anothervalue",
	Run: func(cmd *cobra.Command, args []string) {
		podLabel = make(Labels)
		namespace, _ := cmd.Flags().GetString("namespace")
		elementName, _ := cmd.Flags().GetString("element")
		metricName, _ := cmd.Flags().GetString("metric")
		metricValue, _ := cmd.Flags().GetInt("value")
		metricTag, _ := cmd.Flags().GetString("tag")
		metricLabel, _ := cmd.Flags().GetString("label")
		// Add your logic to push a metric here
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

		// Check if service exists
		isServiceExisting, err := knImpl.IsServiceExisting(elementName, namespace)
		if err != nil {
			println(err)
			return
		}
		// Create service with specific name and namespace
		if !isServiceExisting {
			knImpl.CreateService(namespace, elementName, v1.ServiceTypeClusterIP)
		}

		// Use existing service if

		// Check if serviceMonitor exists
		isServiceMonitorExisting, err := knImpl.IsServiceMonitorExisting(elementName, namespace)
		if err != nil {
			println(err)
			return
		}

		// Create service Monitor
		if !isServiceMonitorExisting {
			knImpl.CreateServiceMonitor(namespace, elementName)
		}
		// Use existing serviceMonitor

		// Check if pod exists
		isPodExisting, err := knImpl.IsPodExisting(elementName, namespace)
		if err != nil {
			println(err)
			return
		}

		// delete pod
		if isPodExisting {
			knImpl.DeletePod(namespace, elementName)
		}
		// Create pod metric generator
		knImpl.CreatePod(namespace, elementName, []string{"/bin/sh", "-c", fmt.Sprintf(`while true; do
									 echo "# HELP %s A custom gauge metric" > /usr/share/nginx/html/metrics;
									 echo "# TYPE %s gauge" >> /usr/share/nginx/html/metrics;
									 echo "%s{%s=\"%s\"} %d" >> /usr/share/nginx/html/metrics;
									 sleep 5;
								   done`, metricName, metricName, metricName, metricLabel, metricTag, metricValue)}, podLabel)

		// create exposing metric pod

		// knImpl.CreatePod(namespace, elementName,)
	},
}
