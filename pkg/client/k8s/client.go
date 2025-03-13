package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeClient holds the Kubernetes clientset and configuration
type KubeClient struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewKubeClient creates and returns a new KubeClient, establishing the connection
func NewKubeClient(opts *Options) (*KubeClient, error) {
	// Convert options to REST config
	config, err := opts.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config: %w", err)
	}

	// Create the Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Return the new KubeClient
	return &KubeClient{
		clientset: clientset,
		config:    config,
	}, nil
}

// GetClientset returns the Kubernetes clientset
func (c *KubeClient) GetClientset() *kubernetes.Clientset {
	return c.clientset
}

// GetConfig returns the Kubernetes REST config
func (c *KubeClient) GetConfig() *rest.Config {
	return c.config
}

// Ping checks if the Kubernetes API server is reachable
func (c *KubeClient) Ping(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to ping Kubernetes API server: %w", err)
	}
	return nil
}
