package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubernetesClient is an interface for mocking purposes
type KubernetesClient interface {
	CreateNamespace(name string) error
	CreatePod(namespace, name string, imageArgs []string) error
	CreateService(namespace, name string, serviceType corev1.ServiceType) error
	CreateServiceMonitor(namespace, name string) error
	IsNamespaceExisting(namespace string) (bool, error)
	IsPodExisting(name, namespace string) (bool, error)
	IsServiceExisting(name, namespace string) (bool, error)
	IsServiceMonitorExisting(name, namespace string) (bool, error)
	DeletePod(name, namespace string) error
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
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := c.clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	return err
}

// CreatePod creates a pod
func (c *Client) CreatePod(namespace, name string, imageArgs []string, labels map[string]string) error {
	unprivileged := false
	readOnly := true
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name:   name,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "metrics",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: "busybox",
					Args:  imageArgs,
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						Privileged:             &unprivileged,
						ReadOnlyRootFilesystem: &readOnly,
					},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      "metrics",
							MountPath: "/usr/share/nginx/html",
						},
					},
				},
				{
					Name:    name + "" + "exposing",
					Image:   "nginx:alpine",
					Command: []string{"/bin/sh", "-c"},
					Args: []string{` # Configure Nginx
          echo 'server {
              listen 80;
              location /metrics {
                  default_type text/plain;
                  add_header Content-Type "text/plain; version=0.0.4; charset=utf-8";
                  root /usr/share/nginx/html;
              }
          }' > /etc/nginx/conf.d/default.conf;
          # Start Nginx
          nginx -g 'daemon off;'`,
					},
					Ports: []corev1.ContainerPort{corev1.ContainerPort{
						ContainerPort: 80,
					}},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/metrics",
								Port: intstr.FromInt(80),
								HTTPHeaders: []corev1.HTTPHeader{
									corev1.HTTPHeader{
										Name:  "Accept",
										Value: "text/plain; version=0.0.4; charset=utf-8",
									},
								},
							},
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/metrics",
								Port: intstr.FromInt(80),
								HTTPHeaders: []corev1.HTTPHeader{
									corev1.HTTPHeader{
										Name:  "Accept",
										Value: "text/plain; version=0.0.4; charset=utf-8",
									},
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      "metrics",
							MountPath: "/usr/share/nginx/html",
						},
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
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
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

func (c *Client) IsNamespaceExisting(namespace string) (bool, error) {
	_, err := c.clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

// IsPodExisting checks if a pod exists in a namespace
func (c *Client) IsPodExisting(name, namespace string) (bool, error) {
	_, err := c.clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

// IsServiceExisting checks if a service exists in a namespace
func (c *Client) IsServiceExisting(name, namespace string) (bool, error) {
	_, err := c.clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

// IsServiceMonitorExisting checks if a ServiceMonitor exists in a namespace
func (c *Client) IsServiceMonitorExisting(name, namespace string) (bool, error) {
	_, err := c.monitoringClientset.MonitoringV1().ServiceMonitors(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func (c *Client) DeletePod(namespace, name string) error {
	err := c.clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	return err
}
