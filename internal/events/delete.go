package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Delete a deployment with the given name in the specified namespace
// Parameters:
// - clientset: Kubernetes clientset used to perform the deletation
// - name: Name of the Deployment to delete
// - namespace: Namespace where the deployment resides
func Deployment_delete(logger *slog.Logger, dynClient *dynamic.DynamicClient, name string, namespace string) {
	deletePolicy := metav1.DeletePropagationForeground

	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	err := dynClient.Resource(gvr).Namespace(namespace).Delete(
		context.TODO(),
		name,
		metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		},
	)

	if err != nil {
		logger.Error(fmt.Sprintf("❌ "+"Failed to Delete a Deployment. Error %s", err.Error()))
	}
}

// Delete a job with the given name in the specified namespace
// Parameters:
// - clientset: Kubernetes dynamic client used to perform the deletation
// - name: Name of the job to delete
// - namespace: Namespace where the job resides
func Job_delete(logger *slog.Logger, dynClient *dynamic.DynamicClient, name string, namespace string) {
	deletePolicy := metav1.DeletePropagationForeground

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gvr := schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}

	err := dynClient.Resource(gvr).Namespace(namespace).Delete(
		ctx,
		name,
		metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		},
	)

	if err != nil {
		logger.Error("❌ " + fmt.Sprintf("Failed to Delete a Job. Name: %s", name))
	}
}
