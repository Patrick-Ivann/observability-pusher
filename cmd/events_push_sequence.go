package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/Patrick-Ivann/observability-pusher/internal/kubernetes"
	"github.com/Patrick-Ivann/observability-pusher/internal/sources"
	"github.com/spf13/cobra"
)

var eventSequenceFilePath string

func init() {
	eventsPushSequenceCmd.Flags().StringVar(&eventSequenceFilePath, "event-sequence-file", os.Getenv("HOME")+"/.obs-pusher/"+"events-sequence.json", "Path to the json file containing the sequence of events")
	eventsPushSequenceCmd.Flags().String("namespace", "default", "Namespace to create the app in")
	eventsPushSequenceCmd.Flags().StringVar(&eventFilePath, "event-file", os.Getenv("HOME")+"/.obs-pusher/"+"events.xml", "Path to the XML file for the event")
	eventsPushSequenceCmd.Flags().Bool("psa-enabled", false, "if the cluster has some Pod Security Admission enabled")

	eventsPushSequenceCmd.Flags().String("registry-path", "", "Registry path for the image")
	eventsPushSequenceCmd.Flags().String("image-pull-secret", "", "Name of the image pull secret")
	eventsPushSequenceCmd.Flags().String("service-account", "default", "Name of the ServiceAccount to use")

}

func generateLogs(sequenceEvents []sources.SequenceNotification, events []sources.Log) string {
	script := "#!/bin/sh\n\n"
	for _, seqEvent := range sequenceEvents {
		isInSlice := slices.ContainsFunc(events, func(c sources.Log) bool { return c.ID == seqEvent.ID })
		if !isInSlice {
			fmt.Printf(" event id %s not in dictionary \n", seqEvent.ID)
			continue
		}
		for i := 0; i < seqEvent.Repetition; i++ {
			for _, event := range events {
				if event.ID == seqEvent.ID {
					println(event.Text)
					formattedMessage := event.Text
					for j, value := range seqEvent.Values {
						placeholder := fmt.Sprintf("{%d}", j)
						formattedMessage = strings.ReplaceAll(formattedMessage, placeholder, value)
					}
					event.Text = formattedMessage

					jsonData, err := json.Marshal(event)
					if err != nil {
						fmt.Println("Error generating JSON:", err)
						continue
					}
					script += fmt.Sprintf("echo \"%s\"\n", string(jsonData))
					script += fmt.Sprintf("sleep %d\n", seqEvent.Interval)
				}
			}
		}
	}
	return script
}

func parseLabels(labelStrings []string) map[string]string {
	labels := make(map[string]string)
	for _, label := range labelStrings {
		parts := strings.Split(label, ":")
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}
	return labels
}

// eventsPushSequenceCmd represents the push command for events
var eventsPushSequenceCmd = &cobra.Command{
	Use:   "push-sequence",
	Short: "Push a sequence of events",
	Long:  "Push a sequence of events",
	Run: func(cmd *cobra.Command, args []string) {
		namespace, _ := cmd.Flags().GetString("namespace")
		isPsaEnabled, _ := cmd.Flags().GetBool("psa-enabled")
		registry, _ := cmd.Flags().GetString("registry-path")
		registryPullSecret, _ := cmd.Flags().GetString("image-pull-secret")
		serviceAccount, _ := cmd.Flags().GetString("service-account")

		// Parse events from JSON files
		eventDictionary, err := sources.ReadDictionary(eventFilePath)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		sequenceEvents, err := sources.ParseSequence(eventSequenceFilePath)
		if err != nil {
			log.Fatalf("Failed to parse sequence events: %v", err)
		}

		// Determine pod name
		podName := "log-generator-pod"
		podLabels.Append(Labels{"obs-pusher": "events"})

		for _, seqEvent := range sequenceEvents {
			if seqEvent.Name != "" {
				podName = seqEvent.Name
			}
			if len(seqEvent.Labels) > 0 {
				seqLabels := parseLabels(seqEvent.Labels)
				podLabels.Append(seqLabels)
			}
		}
		// Generate logs based on the parsed sequence events and events
		script := generateLogs(sequenceEvents, eventDictionary.Logs)

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
		err = knImpl.CreateLogPod(namespace, podName, []string{script}, podLabels, isPsaEnabled)

		if err != nil {
			println(err.Error())
			return
		}

	},
}
