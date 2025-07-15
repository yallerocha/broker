package events

import (
	"context"
	"time"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Delete a deployment with the given name in the specified namespace
// Parameters:
// - dynClient: Kubernetes dynamic client used to perform the deletion
// - name: Name of the Deployment to delete
func Deployment_delete(dynClient *dynamic.DynamicClient, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	namespace := utils.Get_context_namespace()

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

	return err
}

// Delete a job with the given name in the specified namespace
// Parameters:
// - dynClient: Kubernetes dynamic client used to perform the deletion
// - name: Name of the job to delete
func Job_delete(dynClient *dynamic.DynamicClient, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	namespace := utils.Get_context_namespace()

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

	return err
}
