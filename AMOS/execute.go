package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/HarshD0011/AMOS/AMOS/services"
	"github.com/joho/godotenv"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

func Execute() {
	if err := godotenv.Load(); err != nil {
		klog.Warning("No .env file found")
	}

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// Try to build config from flags (kubeconfig path)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		klog.Warningf("Error building kubeconfig from flags: %v. Trying InClusterConfig...", err)
		// Fallback to InClusterConfig
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Errorf("Error building kubernetes config: %v", err)
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("Error building kubernetes client: %s", err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	klog.Info("Ensuring AMOS resources exist...")
	services.Informer(clientset)

	informerDeployment := services.NewInformerDeployment(clientset)
	go informerDeployment.Run(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	klog.Info("Shutting down...")
}
