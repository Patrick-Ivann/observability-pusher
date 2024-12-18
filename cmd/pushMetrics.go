package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/util/homedir"
)

// pushMetricsCmd represents the pushMetrics command
var pushMetricsCmd = &cobra.Command{
	Use:   "push-metrics --metric=<metric_name> --value=<metric_value>",
	Short: "Push metrics to Prometheus Pushgateway",
	Run: func(cmd *cobra.Command, args []string) {
		metric, _ := cmd.Flags().GetString("metric")
		value, _ := cmd.Flags().GetString("value")
		namespace, _ := cmd.Flags().GetString("namespace")

		if metric == "" || value == "" {
			fmt.Println("Please specify both metric and value")
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

		err = isNamespaceExisting(clientset, namespace)
		if err != nil {
			err = createNamespace(clientset, namespace)
			if err != nil {
				fmt.Printf("Error creating namespace: %s\n", err.Error())
				return
			}
		}

		err = createPushgatewayPod(clientset, namespace)
		if err != nil {
			fmt.Printf("Error creating Pushgateway pod: %s\n", err.Error())
			return
		}

		err = createService(clientset, namespace)
		if err != nil {
			fmt.Printf("Error creating service: %s\n", err.Error())
			return
		}
		time.Sleep(2 * time.Second)

		err = portForwardAndPushMetric(config, namespace, metric, value)
		if err != nil {
			fmt.Printf("Error during port forward and metric push: %s\n", err.Error())
			cleanupResources(clientset, namespace)
			return
		}

		defer func() {
			if err := cleanupResources(clientset, namespace); err != nil {
				fmt.Printf("Error during cleanup: %s\n", err.Error())
			}
		}()

		defer fmt.Println("Metric pushed successfully and resources cleaned up")
	},
}

func init() {
	rootCmd.AddCommand(pushMetricsCmd)
	pushMetricsCmd.Flags().String("metric", "", "Name of the metric to push")
	pushMetricsCmd.Flags().String("value", "", "Value of the metric to push")
	pushMetricsCmd.Flags().String("namespace", "default", "Namespace to create resources in")
}

func createNamespace(clientset *kubernetes.Clientset, namespaceName string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	return nil
}

func deletePod(clientset *kubernetes.Clientset, namespace string, podName string) error {

	err := clientset.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
func isPodExisting(clientset *kubernetes.Clientset, namespace string, podName string) error {

	podInfo, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if podInfo.CreationTimestamp.Size() > 0 {
		return nil

	}
	return nil
}

func isNamespaceExisting(clientset *kubernetes.Clientset, namespace string) error {

	namespaceInfo, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if namespaceInfo.CreationTimestamp.Size() > 0 {
		return nil

	}
	return nil
}

func createPushgatewayPod(clientset *kubernetes.Clientset, namespace string) error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pushgateway",
			Labels: map[string]string{
				"app": "pushgateway",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "pushgateway",
					Image: "prom/pushgateway",
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 9091,
						},
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

func createService(clientset *kubernetes.Clientset, namespace string) error {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pushgateway-service",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "pushgateway",
			},
			Ports: []v1.ServicePort{
				{
					Port:     9091,
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}
	_, err := clientset.CoreV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	return nil
}

func portForwardAndPushMetric(config *rest.Config, namespace, metric, value string) error {
	// Create port-forwarder
	var SchemeGroupVersion = schema.GroupVersion{Group: "api", Version: "v1"}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	config.GroupVersion = &SchemeGroupVersion
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}
	req := restClient.Post().Resource("pods").Namespace(namespace).Name("pushgateway").SubResource("portforward")
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return fmt.Errorf("failed to create round tripper: %w", err)
	}
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	defer close(stopChan)
	ports := []string{"9091:9091"}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	fw, err := portforward.New(dialer, ports, stopChan, readyChan, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}
	go func() {
		if err = fw.ForwardPorts(); err != nil {
			fmt.Printf("error in port forwarding: %s\n", err.Error())
		}
	}()
	<-readyChan
	// Wait for the port-forward to be ready
	time.Sleep(2 * time.Second)
	// Push the metric
	pushURL := fmt.Sprintf("http://localhost:9091/metrics/job/some_job")
	data := fmt.Sprintf("%s %s\n", metric, value)
	resp, err := http.Post(pushURL, "text/plain", strings.NewReader(data))

	// read response body
	body, readingError := io.ReadAll(resp.Body)
	if readingError != nil {
		fmt.Println(readingError)
	}
	// close response body
	resp.Body.Close()

	// print response body
	fmt.Println(string(body))

	if err != nil {
		fmt.Println(resp)
		return fmt.Errorf("failed to push metrics: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push metrics, status: %s", resp.Status)
	}
	return nil
}

func cleanupOnInterrupt(clientset *kubernetes.Clientset, namespace string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nReceived interrupt. Cleaning up resources...")
		if err := cleanupResources(clientset, namespace); err != nil {
			fmt.Printf("Error during cleanup: %s\n", err.Error())
		}
		os.Exit(1)
	}()
}

// func portForwardAndPushMetric(config *rest.Config, namespace, metric, value string) error {
// 	cmd := exec.Command("kubectl", "port-forward", "svc/pushgateway-service", "9091:9091", "-n", namespace)
// 	cmd.Env = os.Environ()
// 	stdout, err := cmd.StdoutPipe()
// 	if err != nil {
// 		return fmt.Errorf("failed to get stdout pipe: %w", err)
// 	}
// 	stderr, err := cmd.StderrPipe()
// 	if err != nil {
// 		return fmt.Errorf("failed to get stderr pipe: %w", err)
// 	}
// 	if err := cmd.Start(); err != nil {
// 		return fmt.Errorf("failed to start port-forward command: %w", err)
// 	}

// 	defer cmd.Process.Kill()

// 	go func() {
// 		stdout, _ := ioutil.ReadAll(stdout)
// 		stderr, _ := ioutil.ReadAll(stderr)
// 		fmt.Printf("stdout: %s\n", stdout)
// 		fmt.Printf("stderr: %s\n", stderr)
// 	}()

// 	pushURL := fmt.Sprintf("http://localhost:9091/metrics/job/some_job")
// 	data := fmt.Sprintf("%s %s\n", metric, value)
// 	resp, err := http.Post(pushURL, "text/plain", strings.NewReader(data))
// 	if err != nil {
// 		return fmt.Errorf("failed to push metrics: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusAccepted {
// 		return fmt.Errorf("failed to push metrics, status: %s", resp.Status)
// 	}

// 	return nil
// }

func cleanupResources(clientset *kubernetes.Clientset, namespace string) error {
	err := clientset.CoreV1().Pods(namespace).Delete(context.Background(), "pushgateway", metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	err = clientset.CoreV1().Services(namespace).Delete(context.Background(), "pushgateway-service", metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}
