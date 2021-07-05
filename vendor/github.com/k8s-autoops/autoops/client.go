package autoops

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func InClusterClient() (client *kubernetes.Clientset, err error) {
	var cfg *rest.Config
	if cfg, err = rest.InClusterConfig(); err != nil {
		return
	}
	if client, err = kubernetes.NewForConfig(cfg); err != nil {
		return
	}
	return
}
