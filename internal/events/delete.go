package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/gomorpheus/morpheus-go-sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func Morpheus_Delete_Workload(morpheusClient *utils.MorpheusClient, workload utils.Workload, clusterID int) error {
	deploymentID, err := findDeploymentIdByName(morpheusClient, clusterID, workload.Name)
	if err != nil {
		return fmt.Errorf("failed to find deployment ID for deployment %s in cluster %d: %v", workload.Name, clusterID, err)
	}
	// Use the cluster-scoped endpoint to delete the deployment from the Kubernetes cluster
	resp, err := morpheusClient.Client.Execute(&morpheus.Request{
		Method: "DELETE",
		Path:   fmt.Sprintf("/api/clusters/%d/deployments/%s?namespace=default", clusterID, deploymentID),
	})

	if err != nil {
		return fmt.Errorf("failed to delete deployment %s from cluster %d: %v", deploymentID, clusterID, err)
	}
	if !resp.Success {
		return fmt.Errorf("API request failed: %s", resp.Error)
	}

	return nil
}

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

func findDeploymentIdByName(morpheusClient *utils.MorpheusClient, clusterID int, deploymentName string) (string, error) {
	resp, err := morpheusClient.Client.Execute(&morpheus.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/api/clusters/%d/deployments?namespace=default", clusterID),
	})
	if err != nil {
		log.Printf("[DEBUG] Error listing deployments in cluster %d: %v", clusterID, err)
		return "", fmt.Errorf("failed to list deployments in cluster %d: %v", clusterID, err)
	}
	if !resp.Success {
		log.Printf("[DEBUG] API request failed when listing deployments: %s", resp.Error)
		return "", fmt.Errorf("API request failed when listing deployments: %s", resp.Error)
	}

	// Parse the response to find the deployment with the given name
	var result map[string]interface{}
	if resp.Result != nil {
		result, _ = resp.Result.(map[string]interface{})
	}
	if result == nil && resp.Body != nil {
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			log.Printf("[DEBUG] Failed to unmarshal resp.Body: %v", err)
			return "", fmt.Errorf("invalid response format and failed to unmarshal body when listing deployments")
		}
	}
	if result == nil {
		log.Printf("[DEBUG] No result or body found in response for cluster deployments list")
		return "", fmt.Errorf("no response body found for deployments in cluster %d", clusterID)
	}

	var deployments []interface{}
	if v, ok := result["deployments"]; ok {
		deployments, _ = v.([]interface{})
	}
	if deployments == nil {
		log.Printf("[DEBUG] No deployments array found in response for cluster %d", clusterID)
		return "", fmt.Errorf("no deployments array found in response for cluster %d", clusterID)
	}

	for _, d := range deployments {
		deploy, ok := d.(map[string]interface{})
		if !ok {
			continue
		}
		var name string
		if n, ok := deploy["name"].(string); ok {
			name = n
		}
		if name == deploymentName {
			if id, ok := deploy["id"].(float64); ok {
				return fmt.Sprintf("%v", id), nil
			}
		}
	}
	log.Printf("[WARN] Deployment with name %s not found in cluster %d. Skipping migration for this deployment.", deploymentName, clusterID)
	// Return empty string and nil error to indicate skip
	return "", nil
}
