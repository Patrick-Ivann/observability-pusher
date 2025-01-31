package cmd

import (
	"fmt"

	"github.com/Patrick-Ivann/observability-pusher/internal/kubernetes"

	"github.com/spf13/cobra"
)

func init() {

	metricsPushCmd.Flags().String("namespace", "test", "Namespace to create resources in")
	metricsPushCmd.Flags().String("element", "", "Name of producing app")
	metricsPushCmd.Flags().String("metric", "", "Name of the metric to push")
	metricsPushCmd.Flags().Int("value", 0, "Value of the metric to push")
	metricsPushCmd.Flags().String("tag-value", "", "")
	metricsPushCmd.Flags().String("tag-label", "", "")
	metricsPushCmd.Flags().Bool("psa-enabled", false, "if the cluster has some Pod Security Admission enabled")
	metricsPushCmd.Flags().String("service-account", "default", "Name of the ServiceAccount to use")
	// metricsPushCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)

}

func generateMetricCommand(metricName string, metricValue int, metricTagLabel, metricTagValue string) string {
	if metricTagLabel != "" && metricTagValue != "" {
		return fmt.Sprintf(`while true; do
        echo "# HELP %s A custom gauge metric" > /usr/share/nginx/html/metrics;
        echo "# TYPE %s gauge" >> /usr/share/nginx/html/metrics;
        echo "%s{%s=\"%s\"} %d" >> /usr/share/nginx/html/metrics;
        sleep 5;
        done`, metricName, metricName, metricName, metricTagLabel, metricTagValue, metricValue)
	}

	return fmt.Sprintf(`while true; do
    echo "# HELP %s A custom gauge metric" > /usr/share/nginx/html/metrics;
    echo "# TYPE %s gauge" >> /usr/share/nginx/html/metrics;
    echo "%s %d" >> /usr/share/nginx/html/metrics;
    sleep 5;
    done`, metricName, metricName, metricName, metricValue)
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
		isPsaEnabled, _ := cmd.Flags().GetBool("psa-enabled")
		registry, _ := cmd.Flags().GetString("registry-path")
		registryPullSecret, _ := cmd.Flags().GetString("image-pull-secret")
		serviceAccount, _ := cmd.Flags().GetString("service-account")
		knImpl, err := kubernetes.NewClientset(registry, registryPullSecret, serviceAccount)
		if err != nil {
			println(err.Error())
			return
		}

		// check if namespace exists
		isNamespaceExisting, err := knImpl.IsNamespaceExisting(namespace)
		if err != nil {
			println(err.Error())
			return
		}
		// create namespace
		if !isNamespaceExisting {
			knImpl.CreateNamespace(namespace)
		}

		// Check if pod exists by fetching it based on labels
		services, err := knImpl.FetchServiceByLabels(namespace, Labels{"obs-pusher": "metrics"})
		if err != nil {
			println(err.Error())
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
		servicemonitors, err := knImpl.FetchServiceMonitorByLabels(namespace, Labels{"obs-pusher": "metrics"})
		if err != nil {
			println(err.Error())
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
		podList, err := knImpl.FetchPodByLabels(namespace, Labels{"obs-pusher": "metrics"})
		if err != nil {
			println(err.Error())
			return
		}

		// delete existing pod if it exists
		if len(podList.Items) > 0 {
			for _, pod := range podList.Items {
				knImpl.DeletePod(namespace, pod.Name)
				knImpl.WaitForPodDeletion(namespace, pod.Name)
			}
		}

		// Generate metric command based on provided tags and values
		metricCommand := generateMetricCommand(metricName, metricValue, metricTagLabel, metricTagValue)
		knImpl.CreateMetricPod(namespace, elementName, []string{"/bin/sh", "-c", metricCommand}, podLabels, isPsaEnabled)

	},
}
