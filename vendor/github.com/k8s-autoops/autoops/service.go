package autoops

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ServiceGetOrCreate(ctx context.Context, client *kubernetes.Clientset, service *corev1.Service) (serviceOut *corev1.Service, err error) {
	if serviceOut, err = client.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			serviceOut, err = client.CoreV1().Services(service.Namespace).Create(ctx, service, metav1.CreateOptions{})
		}
	}

	return
}
