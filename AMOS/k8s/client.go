package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/HarshD0011/AMOS/AMOS/config"
	
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes clientset and configuration
type Client struct {
	Clientset *kubernetes.Clientset
	Config    *rest.Config
	Namespaces []string
}

// NewClient creates a new Kubernetes client based on the provided configuration
func NewClient(cfg *config.KubernetesConfig) (*Client, error) {
	var k8sConfig *rest.Config
	var err error

	if cfg.InCluster {
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		kubeconfig := cfg.KubeConfigPath
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			} else {
				kubeconfig = os.Getenv("KUBECONFIG")
			}
		}

		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			// Fallback to in-cluster if kubeconfig fails, helpful for hybrid setups
			k8sConfig, err = rest.InClusterConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to build kubeconfig from %s and in-cluster fallback failed: %w", kubeconfig, err)
			}
		}
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Client{
		Clientset:  clientset,
		Config:     k8sConfig,
		Namespaces: cfg.Namespaces,
	}, nil
}
