package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Patrick-Ivann/observability-pusher/internal/kubernetes"
	"github.com/Patrick-Ivann/observability-pusher/internal/sources"
	"github.com/spf13/cobra"
)

var eventFilePath string
var eventID string

func init() {

	eventsPushFromDictionaryCmd.Flags().String("name", "", "Name of producing app")
	eventsPushFromDictionaryCmd.Flags().String("namespace", "default", "Namespace to create the app in")
	eventsPushFromDictionaryCmd.Flags().String("message", "", "Message to print at regular intervals. IF a value is provided to event-id, this flag will fill the message template e.g '--message=value1,value2'")
	eventsPushFromDictionaryCmd.Flags().Int("interval", 5, "interval")
	eventsPushFromDictionaryCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)
	eventsPushFromDictionaryCmd.Flags().StringVar(&eventFilePath, "event-file", os.Getenv("HOME")+"/.obs-pusher/"+"dictionary.xml", "Path to the XML file for the event")
	eventsPushFromDictionaryCmd.Flags().StringVar(&eventID, "event-id", "", "The ID of the event to generate")
	eventsPushFromDictionaryCmd.Flags().Bool("psa-enabled", false, "if the cluster has some Pod Security Admission enabled")

	eventsPushFromDictionaryCmd.Flags().String("image-pull-secret", "", "Name of the image pull secret")
	eventsPushFromDictionaryCmd.Flags().String("registry-path", "", "Registry path for the image")
	eventsPushFromDictionaryCmd.Flags().String("service-account", "default", "Name of the ServiceAccount to use")

}

var eventsPushFromDictionaryCmd = &cobra.Command{
	Use:   "push-dict",
	Short: "Push an event based on the dictionary provided",
	Long:  "Push an event based on the dictionary provided --name=<> --event-id=<> --message=value1,value2 --namespace=<> --interval=10 --pod-labels=bip:boup",
	Run: func(cmd *cobra.Command, args []string) {

		namespace, _ := cmd.Flags().GetString("namespace")
		applicationName, _ := cmd.Flags().GetString("name")
		message, _ := cmd.Flags().GetString("message")
		intervalInSecond, _ := cmd.Flags().GetInt("interval")
		isPsaEnabled, _ := cmd.Flags().GetBool("psa-enabled")
		registry, _ := cmd.Flags().GetString("registry-path")
		registryPullSecret, _ := cmd.Flags().GetString("image-pull-secret")
		serviceAccount, _ := cmd.Flags().GetString("service-account")

		podLabels.Append(Labels{"obs-pusher": "events"})

		// Use event file and ID if provided
		if eventID != "" {
			dictionary, err := sources.ReadDictionary(eventFilePath)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			var selectedNotification *sources.Log
			for _, notification := range dictionary.Logs {
				if notification.ID == eventID {
					selectedNotification = &notification
					break
				}
			}

			if selectedNotification == nil {
				fmt.Println("Error: Notification with the specified ID not found")
				return
			}

			notifValuesArray := strings.Split(message, ",")
			formattedMessage := selectedNotification.Text
			for i, value := range notifValuesArray {
				placeholder := fmt.Sprintf("{%d}", i)
				formattedMessage = strings.ReplaceAll(formattedMessage, placeholder, value)
			}

			selectedNotification.Text = formattedMessage
			jsonData, err := json.Marshal(selectedNotification)
			if err != nil {
				fmt.Println("Error generating JSON:", err)
				return
			}

			message = string(jsonData)

		}

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
		podList, err := knImpl.FetchPodByLabels(namespace, Labels{"obs-pusher": "events"})
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
		// Create a new pod
		err = knImpl.CreateLogPod(namespace, applicationName, []string{fmt.Sprintf(`while true; do echo '%s'; sleep %d; done`, message, intervalInSecond)}, podLabels, isPsaEnabled)

		if err != nil {
			println(err.Error())
			return
		}

	},
}
