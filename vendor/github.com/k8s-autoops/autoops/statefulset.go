package autoops

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func StatefulSetGetOrCreate(ctx context.Context, client *kubernetes.Clientset, sts *appsv1.StatefulSet) (stsOut *appsv1.StatefulSet, err error) {
	if stsOut, err = client.AppsV1().StatefulSets(sts.Namespace).Get(ctx, sts.Name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			stsOut, err = client.AppsV1().StatefulSets(sts.Namespace).Create(ctx, sts, metav1.CreateOptions{})
		}
	}

	return
}
