package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/HarshD0011/AMOS/AMOS/services"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		klog.Errorf("Error building kubeconfig: %s", err.Error())
		// If explicit kubeconfig failed, try in-cluster config (though BuildConfigFromFlags actually handles in-cluster if url/kubeconfig are empty,
		// but here we might have passed a default path that doesn't exist).
		// For now, fail hard if config fails.
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("Error building kubernetes client: %s", err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the setup/informer function to create resources (if this is what the user wants)
	klog.Info("Running services.Informer to ensure resources exist...")
	services.Informer(clientset)

	// Start PodMonitor
	podMonitor := services.NewPodMonitor(clientset)
	go podMonitor.Run(ctx)
	klog.Info("Started PodMonitor")

	// Start DeploymentMonitor
	deploymentMonitor := services.NewDeploymentMonitor(clientset)
	go deploymentMonitor.Run(ctx)
	klog.Info("Started DeploymentMonitor")

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	klog.Info("Shutting down...")
}
