package cmd

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"path/filepath"

	"github.com/spf13/cobra"
	v1Core "k8s.io/api/core/v1"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type LogMessages struct {
	Messages []Message `xml:"message"`
}

type Message struct {
	Key   string `xml:"key,attr"`
	Value string `xml:",chardata"`
}

var xmlFile string
var numMessages int
var interval int

var pushEventsCmd = &cobra.Command{
	Use:   "push-events",
	Short: "Create a pod that prints an event at regular intervals",
	Run: func(cmd *cobra.Command, args []string) {
		messages := parseXML(xmlFile)
		selectedMessage := selectMessage(messages)

		config, err := rest.InClusterConfig()
		if err != nil {
			home := homedir.HomeDir()
			kubeconfig := filepath.Join(home, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				panic(err.Error())
			}
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		for i := 0; i < numMessages; i++ {
			spawnPod(clientset, selectedMessage)
			time.Sleep(time.Duration(interval) * time.Second)
		}
	},
}

func init() {

	pushEventsCmd.Flags().StringVarP(&xmlFile, "xml", "x", "", "Path to the XML file containing log messages")
	pushEventsCmd.Flags().IntVarP(&numMessages, "num", "n", 1, "Number of log messages to output")
	pushEventsCmd.Flags().IntVarP(&interval, "interval", "i", 1, "Interval between log messages in seconds")
	pushEventsCmd.MarkFlagRequired("xml")

	if err := pushEventsCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// func init() {
// 	rootCmd.AddCommand(pushLogsCmd)
// 	pushLogsCmd.Flags().String("namespace", "default", "Namespace to create resources in")
// 	pushLogsCmd.Flags().String("message", "", "Message to print at regular intervals")
// 	pushLogsCmd.Flags().Int("interval", 5, "interval")

// 	pushLogsCmd.AddCommand(generateDummyDataCmd)
// 	generateDummyDataCmd.Flags().String("model", "", "JSON model to generate dummy data from")
// }

func parseXML(filePath string) map[string]string {
	xmlData, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading XML file: %v\n", err)
		os.Exit(1)
	}

	var logMessages LogMessages
	err = xml.Unmarshal(xmlData, &logMessages)
	if err != nil {
		fmt.Printf("Error unmarshalling XML: %v\n", err)
		os.Exit(1)
	}

	messages := make(map[string]string)
	for _, message := range logMessages.Messages {
		messages[message.Key] = message.Value
	}
	return messages
}

func selectMessage(messages map[string]string) string {
	fmt.Println("Available log messages:")
	for key := range messages {
		fmt.Println(key)
	}

	var selectedKey string
	fmt.Print("Select a message: ")
	fmt.Scan(&selectedKey)

	message, exists := messages[selectedKey]
	if !exists {
		fmt.Printf("Message not found for key: %s\n", selectedKey)
		os.Exit(1)
	}
	return message
}

func spawnPod(clientset *kubernetes.Clientset, logMessage string) {

	pod := &v1Core.Pod{
		ObjectMeta: v1Meta.ObjectMeta{
			Name: "log-pod",
		},
		Spec: v1Core.PodSpec{
			Containers: []v1Core.Container{
				{
					Name:  "log-container",
					Image: "alpine:latest",
					Command: []string{
						"sh", "-c", fmt.Sprintf("for i in $(seq 1 %d); do echo \"%s\"; sleep %d; done",
							1, logMessage, 1),
					},
				},
			},
		},
	}

	_, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, v1Meta.CreateOptions{})
	if err != nil {
		fmt.Printf("Error creating pod: %v\n", err)
		os.Exit(1)
	}
}
