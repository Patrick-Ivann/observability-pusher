package kubernetes

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

type MockClient struct {
	Namespaces map[string]bool
}

func NewMockClient() *MockClient {
	return &MockClient{
		Namespaces: make(map[string]bool),
	}
}

func (m *MockClient) CreateNamespace(name string) error {
	if _, exists := m.Namespaces[name]; exists {
		return errors.New("namespace already exists")
	}
	m.Namespaces[name] = true
	return nil
}

func (m *MockClient) CreatePod(namespace, name, image string, command []string) error {
	return nil
}

func (m *MockClient) CreateService(namespace, name string, serviceType corev1.ServiceType) error {
	return nil
}

func (m *MockClient) CreateServiceMonitor(namespace, name string) error {
	return nil
}
