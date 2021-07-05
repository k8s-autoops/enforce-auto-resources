package autoops

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

func GenerateCert(template, parent *x509.Certificate, publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) (certOut *x509.Certificate, certPEM []byte, err error) {
	var certRaw []byte
	if certRaw, err = x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey); err != nil {
		return
	}

	if certOut, err = x509.ParseCertificate(certRaw); err != nil {
		return
	}

	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certRaw,
	})

	return
}

func GenerateRootCA() (certOut *x509.Certificate, certPEM []byte, keyOut *rsa.PrivateKey, keyPEM []byte, err error) {
	var template = &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"AutoOps"},
			CommonName:   "AutoOps Common Root CA",
		},
		NotBefore:             time.Now().Add(-10 * time.Second),
		NotAfter:              time.Now().AddDate(30, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	if keyOut, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
		return
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(keyOut),
	})
	if certOut, certPEM, err = GenerateCert(template, template, &keyOut.PublicKey, keyOut); err != nil {
		return
	}
	return
}

func GenerateServerCert(names []string, caCertPEM, caKeyPEM []byte) (certOut *x509.Certificate, certPEM []byte, keyOut *rsa.PrivateKey, keyPEM []byte, err error) {
	if len(names) == 0 {
		err = fmt.Errorf("missing %s", "names")
		return
	}
	var (
		caCert *x509.Certificate
		caKey  *rsa.PrivateKey
	)
	{
		caCertRaw, _ := pem.Decode(caCertPEM)
		if caCertRaw == nil {
			err = fmt.Errorf("invalid %s", "caCertPEM")
			return
		}
		if caCertRaw.Type != "CERTIFICATE" {
			err = fmt.Errorf("invalid caCertPEM type: %s", caCertRaw.Type)
			return
		}
		if caCert, err = x509.ParseCertificate(caCertRaw.Bytes); err != nil {
			return
		}
	}
	{
		caKeyRaw, _ := pem.Decode(caKeyPEM)
		if caKeyRaw == nil {
			err = fmt.Errorf("invalid %s", "caKeyPEM")
			return
		}
		if caKeyRaw.Type != "RSA PRIVATE KEY" {
			err = fmt.Errorf("invalid caKeyPEM type: %s, only ", caKeyRaw.Type)
			return
		}
		if caKey, err = x509.ParsePKCS1PrivateKey(caKeyRaw.Bytes); err != nil {
			return
		}
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"AutoOps"},
			CommonName:   names[0],
		},
		NotBefore:      time.Now().Add(-10 * time.Second),
		NotAfter:       time.Now().AddDate(30, 0, 0),
		KeyUsage:       x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:           false,
		MaxPathLenZero: true,
		DNSNames:       names[1:],
	}
	if keyOut, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
		return
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(keyOut),
	})
	if certOut, certPEM, err = GenerateCert(template, caCert, &keyOut.PublicKey, caKey); err != nil {
		return
	}
	return
}
