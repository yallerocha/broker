package eventhandler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cloud-ai-ufcg/broker/internal/events"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"k8s.io/client-go/dynamic"
)

// Deployment_action identifies the specific action for a Kubernetes Deployment and performs it.
// It receives a dynamic client (connected to Karmada for propagation), a DataFrame
// containing the event data, and the index of the current Deployment event.
func Deployment_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
	replicas, _ := df.Col("replicas").Elem(idx).Int()
	label := df.Col("label").Elem(idx).String() // Label is read for logging/information, propagated by Karmada.

	// --- Memory Logic for Deployment ---
	// Attempts to parse memory as a float for normalized values (e.g., "0.08").
	// If parsing fails (indicating a string with a unit like "256Mi"), it uses the raw string.
	memInput := df.Col("memory").Elem(idx).String()
	var memRequested string
	if memFloat, err := strconv.ParseFloat(memInput, 64); err == nil {
		// If it's a number (normalized value), convert to MiB (0.08 * 1024 = 81.92 MiB).
		memRequested = fmt.Sprintf("%dMi", int64(memFloat*1024))
	} else {
		// If it's a K8s format string (e.g., "256Mi"), use it as is.
		memRequested = memInput
	}

	// --- CPU Logic for Deployment (Unit corresponds to a full core) ---
	// The CPU string is used directly. Expects input like "X" (for X cores) or "Xm" (for X millicores).
	// Kubernetes will interpret "4" as 4 cores, "250m" as 250 millicores.
	cpuRequested := df.Col("cpu").Elem(idx).String()

	deployment := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: cpuRequested,
		MemRequested: memRequested,
		Label:        label,
		Annotations:  map[string]string{}, // Karmada-specific annotations are added in update/create logic.
	}

	action := strings.ToLower(df.Col("action").Elem(idx).String())

	// Log that the action will be propagated by Karmada based on the label.
	utils.Log_info(fmt.Sprintf("➡️ [%ss] [Deployment] %s: %s (propagated by Karmada for label '%s')", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), deployment.Name, label))
	switch action {
	case "create":
		err := events.Create_Workload(dynamicContext, deployment, "apps", "deployments")
		utils.Log_err("Failed to create Deployment", err)
	case "delete":
		err := events.Deployment_delete(dynamicContext, deployment.Name)
		utils.Log_err("Failed to delete Deployment", err)
	case "update":
		err := events.Deployment_update(dynamicContext, deployment)
		utils.Log_err("Failed to update Deployment", err)
	default:
		utils.Log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf(""))
	}
}

// Job_action identifies the specific action for a Kubernetes Job and performs it.
// It receives a dynamic client (connected to Karmada for propagation), a DataFrame
// containing the event data, and the index of the current Job event.
func Job_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
	replicas, _ := df.Col("replicas").Elem(idx).Int()
	label := df.Col("label").Elem(idx).String() // Label is read for logging/information, propagated by Karmada.

	// --- Memory Logic for Job ---
	// Same logic as Deployment: detects if it's a float (normalized) or a K8s unit string.
	memInput := df.Col("memory").Elem(idx).String()
	var memRequested string
	if memFloat, err := strconv.ParseFloat(memInput, 64); err == nil {
		memRequested = fmt.Sprintf("%dMi", int64(memFloat*1024))
	} else {
		memRequested = memInput
	}

	// --- CPU Logic for Job (Unit corresponds to a millicore) ---
	// Attempts to parse CPU as an integer for numerical values (e.g., "4").
	// If parsing fails (indicating a string with a unit like "250m"), it uses the raw string.
	cpuInput := df.Col("cpu").Elem(idx).String()
	var cpuRequested string
	if cpuInt, err := strconv.ParseInt(cpuInput, 10, 64); err == nil {
		// If it's a number (normalized value), convert to millicores (e.g., "4" becomes "4m").
		cpuRequested = fmt.Sprintf("%dm", cpuInt)
	} else {
		// If it's a K8s format string (e.g., "250m"), use it as is.
		cpuRequested = cpuInput
	}

	job := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: cpuRequested,
		MemRequested: memRequested,
		Label:        label,
		Annotations: map[string]string{
			"pod-complete.stage.kwok.x-k8s.io/delay": df.Col("job_duration").Elem(idx).String() + "s", // Kwok-specific annotation for job duration.
		},
	}

	action := strings.ToLower(df.Col("action").Elem(idx).String())

	// Log that the action will be propagated by Karmada based on the label.
	utils.Log_info(fmt.Sprintf("➡️ [%ss] [Job] %s: %s (propagated by Karmada for label '%s')", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), job.Name, label))
	switch action {
	case "create":
		err := events.Create_Workload(dynamicContext, job, "batch", "jobs")
		utils.Log_err("Failed to create Job", err)
	case "delete":
		err := events.Job_delete(dynamicContext, job.Name)
		utils.Log_err("Failed to delete Job", err)
	case "update":
		err := events.Job_update(dynamicContext, job)
		utils.Log_err("Failed to update Job", err)
	default:
		utils.Log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf(""))
	}
}

// Morpheus_Action identifies the specific action for a workload in Morpheus and performs it.
// It receives a Morpheus client, a DataFrame containing the event data, and the index of the current event.
func Morpheus_Action(morpheusClient *utils.MorpheusClient, df dataframe.DataFrame, idx int) {
	replicas, _ := df.Col("replicas").Elem(idx).Int()
	label := df.Col("label").Elem(idx).String()
	clusterID, _ := df.Col("label").Elem(idx).Int()

	memInput := df.Col("memory").Elem(idx).String()
	var memRequested string
	if memFloat, err := strconv.ParseFloat(memInput, 64); err == nil {
		memRequested = fmt.Sprintf("%dMi", int64(memFloat*1024))
	} else {
		memRequested = memInput
	}

	cpuRequested := df.Col("cpu").Elem(idx).String()

	workload := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: cpuRequested,
		MemRequested: memRequested,
		Label:        label,
		Annotations:  map[string]string{}, // Annotations can be added if needed.
	}

	action := strings.ToLower(df.Col("action").Elem(idx).String())

	// Log the action being performed in Morpheus.
	utils.Log_info(fmt.Sprintf("➡️ [%ss] [Morpheus Workload] %s: %s (managed by Morpheus for cluster ID '%s')", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), workload.Name, label))
	switch action {
	case "create":
		err := events.Morpheus_Create_Workload(morpheusClient, workload)
		if err == nil {
			utils.Log_info(fmt.Sprintf("Successfully created deployment %s in cluster %d", workload.Name, clusterID))
		}
		utils.Log_err("Failed to create Morpheus Workload", err)
	case "delete":
		err := events.Morpheus_Delete_Workload(morpheusClient, workload, clusterID)
		if err == nil {
			utils.Log_info(fmt.Sprintf("Successfully deleted deployment %s from cluster %d", workload.Name, clusterID))
		}
		utils.Log_err("Failed to delete Morpheus Workload", err)
	case "update":
		err := events.Morpheus_Update_Workload(morpheusClient, workload, clusterID)
		if err == nil {
			utils.Log_info(fmt.Sprintf("Successfully updated deployment %s in cluster %d", workload.Name, clusterID))
		}
		utils.Log_err("Failed to update Morpheus Workload", err)
	default:
		utils.Log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf(""))
	}
}
