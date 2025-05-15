package events

import (
	"context"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Delete a deployment with the given name in the specified namespace
// Parameters:
// - clientset: Kubernetes clientset used to perform the deletation
// - name: Name of the Deployment to delete
// - namespace: Namespace where the deployment resides
func Deployment_delete(clientset *kubernetes.Clientset, name string, namespace string) {
	deletePolicy := metav1.DeletePropagationForeground

	err := clientset.AppsV1().Deployments(namespace).Delete(
		context.TODO(),
		name,
		metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		},
	)

	if err != nil {
		log.Fatalf("Failed to delete Deployment '%s': %v", name, err)
	}
}

// Delete a deployment with the given name in the specified namespace
// Parameters:
// - clientset: Kubernetes clientset used to perform the deletation
// - name: Name of the Deployment to delete
// - namespace: Namespace where the deployment resides
func Job_delete(clientset *kubernetes.Clientset, name string, namespace string) {
	deletePolicy := metav1.DeletePropagationForeground

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := clientset.BatchV1().Jobs(namespace).Delete(
		ctx,
		name,
		metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		},
	)

	if err != nil {
		log.Fatalf("Failed to delete Job '%s': %v", name, err)
	}
}

