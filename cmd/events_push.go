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
	eventsPushCmd.Flags().String("element", "", "Name of producing app")
	eventsPushCmd.Flags().String("namespace", "default", "Namespace to create the app in")
	eventsPushCmd.Flags().String("message", "", "Message to print at regular intervals. IF a value is provided to event-id, this flag will fill the message template e.g '--message=value1,value2'")
	eventsPushCmd.Flags().Int("interval", 5, "interval")
	eventsPushCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)
	eventsPushCmd.Flags().StringVar(&eventFilePath, "event-file", os.Getenv("HOME")+"/.obs-pusher/"+"dictionary.xml", "Path to the XML file for the event")
	eventsPushCmd.Flags().StringVar(&eventID, "event-id", "", "The ID of the event to generate")
	eventsPushCmd.Flags().Bool("psa-enabled", false, "if the cluster has some Pod Security Admission enabled")

	eventsPushCmd.Flags().String("image-pull-secret", "", "Name of the image pull secret")
	eventsPushCmd.Flags().String("registry-path", "", "Registry path for the image")

}

// eventsPushCmd represents the push command for events
var eventsPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push an event",
	Long:  "Push an event --element <> --event-id=<> --message=something --namespace=<> --interval=10 --pod-labels=bip:boup",
	Run: func(cmd *cobra.Command, args []string) {

		namespace, _ := cmd.Flags().GetString("namespace")
		elementName, _ := cmd.Flags().GetString("element")
		message, _ := cmd.Flags().GetString("message")
		intervalInSecond, _ := cmd.Flags().GetInt("interval")
		isPsaEnabled, _ := cmd.Flags().GetBool("psa-enabled")
		registry, _ := cmd.Flags().GetString("registry-path")
		registryPullSecret, _ := cmd.Flags().GetString("image-pull-secret")
		podLabels.Append(Labels{"obs-pusher": "events"})

		// Use event file and ID if provided
		if eventID != "" {
			dictionary, err := sources.ReadDictionary(eventFilePath)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			var selectedNotification *sources.Notification
			for _, notification := range dictionary.Notifications {
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

		knImpl, err := kubernetes.NewClientset(registry, registryPullSecret)
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
		err = knImpl.CreateLogPod(namespace, elementName, []string{fmt.Sprintf(`while true; do echo '%s'; sleep %d; done`, message, intervalInSecond)}, podLabels, isPsaEnabled)

		if err != nil {
			println(err.Error())
			return
		}
	},
}
