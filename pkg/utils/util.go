package utils

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubepath  string
	namespace string
)

// Set the kubernetes context path
func Set_context_path(path string) {
	kubepath = filepath.Join(
		os.Getenv("HOME"), ".kube", path,
	)
}

// Set the target namespace
func Set_context_namespace(namespace_config string) {
	namespace = namespace_config
}

// Get the target namespace
func Get_context_namespace() string {
	return namespace
}

// Get the current kube context, it must be a valid context to submit
// the actions.
// receives a string that represents the context defined in the config file.
func GetDynamicContext() (*dynamic.DynamicClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubepath)
	if err != nil {
		return nil, err
	}

	config.QPS = 100
	config.Burst = 200

	// Create the dynamic client
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynClient, nil
}
