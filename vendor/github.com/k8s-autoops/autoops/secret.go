package autoops

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KeyPairOptions struct {
	CACertPEM []byte
	CAKeyPEM  []byte
	DNSNames  []string
}

func GenerateKeyPair(opts KeyPairOptions) (certPEM, keyPEM []byte, err error) {
	if len(opts.CACertPEM) == 0 || len(opts.CAKeyPEM) == 0 {
		_, certPEM, _, keyPEM, err = GenerateRootCA()
	} else {
		_, certPEM, _, keyPEM, err = GenerateServerCert(opts.DNSNames, opts.CACertPEM, opts.CAKeyPEM)
	}
	return
}

func EnsureSecretAsKeyPair(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace string,
	name string,
	opts KeyPairOptions,
) (
	certPEM []byte,
	keyPEM []byte,
	err error,
) {
	var secret *corev1.Secret
	if secret, err = client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			err = nil

			if certPEM, keyPEM, err = GenerateKeyPair(opts); err != nil {
				return
			}

			if _, err = client.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Type: corev1.SecretTypeTLS,
				StringData: map[string]string{
					corev1.TLSCertKey:       string(certPEM),
					corev1.TLSPrivateKeyKey: string(keyPEM),
				},
			}, metav1.CreateOptions{}); err != nil {
				return
			}
			return
		} else {
			return
		}
	} else {
		certPEM, keyPEM = secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey]
		if len(certPEM) == 0 {
			err = fmt.Errorf("missing key: %s", corev1.TLSCertKey)
			return
		}
		if len(keyPEM) == 0 {
			err = fmt.Errorf("missing key: %s", corev1.TLSPrivateKeyKey)
			return
		}
	}
	return
}
