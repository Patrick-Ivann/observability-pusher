package cmd

import (
	"fmt"
	"strings"

	"github.com/Patrick-Ivann/observability-pusher/internal/kubernetes"
	"github.com/Patrick-Ivann/observability-pusher/internal/sources"

	"github.com/spf13/cobra"
)

func init() {
	podLabels = Labels{"obs-pusher": "metrics"} // Initialize with default label

	metricsPushDictionaryCmd.Flags().String("metric", "", "Name of the metric to push")
	metricsPushDictionaryCmd.Flags().Int("value", 0, "Value of the metric to push")
	metricsPushDictionaryCmd.Flags().String("tag-value", "", "If the dictionary contains tags to set, provide value using this flag")
	metricsPushDictionaryCmd.Flags().Bool("psa-enabled", false, "if the cluster has some Pod Security Admission enabled")
	metricsPushDictionaryCmd.Flags().String("service-account", "default", "Name of the ServiceAccount to use")

}

// metricsPushDictionaryCmd
var metricsPushDictionaryCmd = &cobra.Command{
	Use:     "push-from",
	Short:   "Push a metric using the dictionary",
	Example: "--metric=<> --value=<> --tag-value=<> --pod-labels=key:value,anotherkey:anothervalue",

	Run: func(cmd *cobra.Command, args []string) {
		metricName, _ := cmd.Flags().GetString("metric")
		metricValue, _ := cmd.Flags().GetInt("value")
		metricTagValue, _ := cmd.Flags().GetString("tag-value")
		isPsaEnabled, _ := cmd.Flags().GetBool("psa-enabled")
		registry, _ := cmd.Flags().GetString("registry-path")
		registryPullSecret, _ := cmd.Flags().GetString("image-pull-secret")
		serviceAccount, _ := cmd.Flags().GetString("service-account")

		knImpl, err := kubernetes.NewClientset(registry, registryPullSecret, serviceAccount)
		if err != nil {
			println(err.Error())
			return
		}

		// Read metrics from the XML file if provided
		if metricName != "" {

			var selectedMetric *sources.Metric
			var namespace string
			var applicationName string

			dictionary, err := sources.ReadDictionary(metricFilePath)
			if err != nil {
				println(err.Error())
				return
			}

			for _, metric := range dictionary.Metrics {
				if metric.Name == metricName {
					selectedMetric = &metric
					break
				}
			}

			if selectedMetric == nil {
				println("Error: Metric with the specified name not found")
				return
			}

			namespace = strings.Split(selectedMetric.Name, ".")[0]
			applicationName = namespace

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
			services, err := knImpl.FetchServiceByLabels(namespace, Labels{"obs-pusher": "metrics", "element": applicationName})
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
			knImpl.CreateService(namespace, applicationName, Labels{"obs-pusher": "metrics", "element": applicationName})

			// Check if pod exists by fetching it based on labels
			servicemonitors, err := knImpl.FetchServiceMonitorByLabels(namespace, Labels{"obs-pusher": "metrics", "element": applicationName})
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

			knImpl.CreateServiceMonitor(namespace, applicationName, Labels{"obs-pusher": "metrics", "element": applicationName})

			// Use existing serviceMonitor

			// Check if pod exists by fetching it based on labels
			podList, err := knImpl.FetchPodByLabels(namespace, Labels{"obs-pusher": "metrics", "element": applicationName})
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

			// Prepare tag values
			tagValuesArray := strings.Split(metricTagValue, ",")
			tags := strings.Split(selectedMetric.Tags, "=")
			valuesMap := make(map[string]string)
			for i, tag := range tags {
				if i < len(tagValuesArray) {
					valuesMap[tag] = tagValuesArray[i]
				} else {
					valuesMap[tag] = ""
				}
			}

			metricTemplate := selectedMetric.GenerateMetricTemplate(valuesMap, metricValue)
			// Generate metric template and create pod metric generator
			metricCommand := fmt.Sprintf(`while true; do
                echo "# HELP %s %s" > /usr/share/nginx/html/metrics;
                echo "# TYPE %s %s" >> /usr/share/nginx/html/metrics;
                %s
                sleep 5;
                done`, selectedMetric.FullyQualifiedName, selectedMetric.Description, selectedMetric.FullyQualifiedName, selectedMetric.Type, metricTemplate)

			knImpl.CreateMetricPod(namespace, applicationName, []string{"/bin/sh", "-c", metricCommand}, podLabels, isPsaEnabled)
			return
		}

	},
}
