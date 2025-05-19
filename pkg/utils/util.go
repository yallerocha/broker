package utils

import (
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Gets the current kube context, it must be a valid context to submit
// the actions.
// receives a string that represents the context defined in the config file.
func GetClientSet(kubetype string) (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", kubetype,
	)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func Exit_if_err(err error, msg string) {
	if err != nil {
		log.Fatalln(msg + ":\n" + err.Error())
	}
}
