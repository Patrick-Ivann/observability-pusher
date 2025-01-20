package kubernetes

import (
	"testing"
)

func TestCreateNamespace(t *testing.T) {
	mockClient := NewMockClient()

	// Test creating a new namespace
	err := mockClient.CreateNamespace("test-namespace")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test creating an existing namespace
	err = mockClient.CreateNamespace("test-namespace")
	if err == nil {
		t.Errorf("expected error, got nil")
	} else if err.Error() != "namespace already exists" {
		t.Errorf("expected 'namespace already exists' error, got %v", err)
	}
}
