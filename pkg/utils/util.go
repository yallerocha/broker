package utils

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubepath string

// Gets the current kube context, it must be a valid context to submit
// the actions.
// receives a string that represents the context defined in the config file.
func GetClientSet(kubetype string) (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", kubetype,
	)

	kubepath = kubeconfig

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		return nil, err
	}

	config.QPS = 100
	config.Burst = 200

	return kubernetes.NewForConfig(config)
}

// Gets the current kube context, it must be a valid context to submit
// the actions.
// receives a string that represents the context defined in the config file.
func GetDynamicContext(logger *slog.Logger) *dynamic.DynamicClient {
	config, err := clientcmd.BuildConfigFromFlags("", kubepath)
	if err != nil {
		logger.Error("Failed to get the config.")
	}

	// Create the dynamic client
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Error("Failed to create the config.")
	}

	return dynClient
}

func Exit_if_err(err error, msg string) {
	if err != nil {
		log.Fatalln(msg + ":\n" + err.Error())
	}
}
