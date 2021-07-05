package autoops

import (
	"bytes"
	"io/ioutil"
	"sync"
)

const (
	PathServiceAccountNamespace = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

var (
	currentNamespace struct {
		Value string
		Err   error
		Once  *sync.Once
	}
)

func init() {
	currentNamespace.Once = &sync.Once{}
}

func CurrentNamespace() (string, error) {
	currentNamespace.Once.Do(func() {
		buf, err := ioutil.ReadFile(PathServiceAccountNamespace)
		currentNamespace.Value, currentNamespace.Err = string(bytes.TrimSpace(buf)), err
	})
	return currentNamespace.Value, currentNamespace.Err
}
