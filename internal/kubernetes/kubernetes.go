package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubernetesClient is an interface for mocking purposes
type KubernetesClient interface {
	CreateNamespace(name string) error
	CreatePod(namespace, name, image string, command []string) error
	CreateService(namespace, name string, serviceType corev1.ServiceType) error
}

// Client implements the KubernetesClient interface
type Client struct {
	clientset           *kubernetes.Clientset
	monitoringClientset *monitoringclient.Clientset
}

// NewClientset creates a new Kubernetes clientset
func NewClientset() (*Client, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		home := homedir.HomeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	monitoringClientset, err := monitoringclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{clientset: clientset, monitoringClientset: monitoringClientset}, nil
}

// CreateNamespace creates a namespace
func (c *Client) CreateNamespace(name string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := c.clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	return err
}

// CreatePod creates a pod
func (c *Client) CreatePod(namespace, name, image string, command []string, labels map[string]string) error {
	unprivileged := false
	readOnly := true
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name:   name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    name,
					Image:   image,
					Command: command,
					SecurityContext: &v1.SecurityContext{
						Capabilities: &v1.Capabilities{
							Drop: []v1.Capability{"ALL"},
						},
						Privileged:             &unprivileged,
						ReadOnlyRootFilesystem: &readOnly,
					},
				},
			},
		},
	}

	_, err := c.clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	return err
}

// CreateService creates a service
func (c *Client) CreateService(namespace, name string, serviceType corev1.ServiceType) error {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []v1.ServicePort{
				{
					Protocol: v1.ProtocolTCP,
					Port:     80,
				},
			},
			Type: serviceType,
		},
	}

	_, err := c.clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	return err
}

// CreateServiceMonitor creates or updates a ServiceMonitor
func (c *Client) CreateServiceMonitor(namespace, name string) error {
	serviceMonitor := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "my-app"},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "http-metrics",
					Path: "/metrics",
				},
			},
		},
	}

	existingServiceMonitor, err := c.monitoringClientset.MonitoringV1().ServiceMonitors(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err := c.monitoringClientset.MonitoringV1().ServiceMonitors(namespace).Create(context.TODO(), serviceMonitor, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating ServiceMonitor: %w", err)
		}
		fmt.Println("ServiceMonitor created successfully")
	} else if err == nil {
		serviceMonitor.ResourceVersion = existingServiceMonitor.ResourceVersion
		_, err := c.monitoringClientset.MonitoringV1().ServiceMonitors(namespace).Update(context.TODO(), serviceMonitor, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("error updating ServiceMonitor: %w", err)
		}
		fmt.Println("ServiceMonitor updated successfully")
	} else {
		return fmt.Errorf("error getting ServiceMonitor: %w", err)
	}

	return nil
}
