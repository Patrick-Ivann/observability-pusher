package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// pushLogsCmd represents the pushLogs command
var pushLogsCmd = &cobra.Command{
	Use:   "push-logs --namespace=<namespace> --message=<message>",
	Short: "Create a pod that prints a message at regular intervals",
	Run: func(cmd *cobra.Command, args []string) {
		namespace, _ := cmd.Flags().GetString("namespace")
		message, _ := cmd.Flags().GetString("message")
		intervalInSecond, _ := cmd.Flags().GetInt("interval")

		POD_NAME := "message-gen"

		if message == "" {
			fmt.Println("Please specify a message")
			return
		}

		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			} else {
				fmt.Println("KUBECONFIG not set and cannot determine home directory")
				return
			}
		}

		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Printf("Error building kubeconfig: %s\n", err.Error())
			return
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Printf("Error creating Kubernetes client: %s\n", err.Error())
			return
		}

		// err = isNamespaceExisting(clientset, namespace)
		// if err != nil {
		// 	err = createNamespace(clientset, namespace)
		// 	if err != nil {
		// 		fmt.Printf("Error creating namespace: %s\n", err.Error())
		// 		return
		// 	}
		// }

		// err = isPodExisting(clientset, namespace, POD_NAME)

		// if err == nil {
		// 	deletePod(clientset, namespace, POD_NAME)
		// 	time.Sleep(time.Second * 2)
		// }

		err = pushLogs(clientset, namespace, POD_NAME, message, intervalInSecond)
		if err != nil {
			fmt.Printf("Error creating message pod: %s\n", err.Error())
			return
		}

		fmt.Println("Message pod created successfully")
	},
}

var generateDummyDataCmd = &cobra.Command{
	Use:   "generate-dummy-data --model=<json_model>",
	Short: "Generate dummy data based on a JSON model",
	Run: func(cmd *cobra.Command, args []string) {
		model, _ := cmd.Flags().GetString("model")

		if model == "" {
			fmt.Println("Please specify a JSON model")
			return
		}

		var jsonModel map[string]interface{}
		err := json.Unmarshal([]byte(model), &jsonModel)
		if err != nil {
			fmt.Printf("Error parsing JSON model: %s\n", err.Error())
			return
		}

		dummyData := generateDummyData(jsonModel)
		dummyDataJSON, err := json.MarshalIndent(dummyData, "", "  ")
		if err != nil {
			fmt.Printf("Error generating dummy data: %s\n", err.Error())
			return
		}

		fmt.Println(string(dummyDataJSON))
	},
}

func init() {
	rootCmd.AddCommand(pushLogsCmd)
	pushLogsCmd.Flags().String("namespace", "default", "Namespace to create resources in")
	pushLogsCmd.Flags().String("message", "", "Message to print at regular intervals")
	pushLogsCmd.Flags().Int("interval", 5, "interval")

	pushLogsCmd.AddCommand(generateDummyDataCmd)
	generateDummyDataCmd.Flags().String("model", "", "JSON model to generate dummy data from")
}

func pushLogs(clientset *kubernetes.Clientset, namespace string, podName string, message string, intervalInSecond int) error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"app": podName,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "message-container",
					Image: "busybox",
					Command: []string{
						"sh", "-c", fmt.Sprintf("while true; do echo '%s'; sleep '%d'; done", message, intervalInSecond),
					},
				},
			},
		},
	}
	_, err := clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	return nil
}

func generateDummyData(model map[string]interface{}) map[string]interface{} {
	dummyData := make(map[string]interface{})
	for key, val := range model {
		switch val.(type) {
		case string:
			dummyData[key] = "dummy_string"
		case int, float64:
			dummyData[key] = rand.Intn(100)
		case bool:
			dummyData[key] = rand.Intn(2) == 1
		case []interface{}:
			dummyData[key] = []interface{}{"dummy_element"}
		case map[string]interface{}:
			dummyData[key] = generateDummyData(val.(map[string]interface{}))
		default:
			dummyData[key] = "unknown_type"
		}
	}
	return dummyData
}
