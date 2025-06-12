package eventhandler

import (
	"fmt"
	"strings"

	"github.com/cloud-ai-ufcg/broker/internal/events"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"k8s.io/client-go/dynamic"
)

// Execute the action required for each deployment
// Receives the dynamicContext for requests,
// a df containing the data and an idx that represents the index of this workload.
func Deployment_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
	replicas, _ := df.Col("replicas").Elem(idx).Int()
	mem_formated := int64(df.Col("memory").Elem(idx).Float() * 1024)

	deployment := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: df.Col("cpu").Elem(idx).String(),
		MemRequested: fmt.Sprintf("%dMi", mem_formated),
		Label:        df.Col("label").Elem(idx).String(),
		Annotations:  map[string]string{},
	}

	action := df.Col("action").Elem(idx).String()
	action = strings.ToLower(action)

	utils.Log_info(fmt.Sprintf("➡️ [%ss] [Deployment] %s: %s", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), deployment.Name))
	if action == "create" {
		err := events.Create_Workload(dynamicContext, deployment, "apps", "deployments")

		utils.Log_err("Failed to create Deployment", err)
	} else if action == "delete" {
		err := events.Deployment_delete(dynamicContext, deployment.Name)

		utils.Log_err("Failed to delete Deployment", err)
	} else if action == "update" {
		err := events.Deployment_update(dynamicContext, deployment)

		utils.Log_err("Failed to update Deployment", err)
	} else {
		utils.Log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf(""))
	}

}

// Execute the action required for each job
// Receives the dynamicContext for requests,
// a df containing the data and an idx that represents the index of this workload.
func Job_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
	replicas, _ := df.Col("replicas").Elem(idx).Int()
	mem_formated := int64(df.Col("memory").Elem(idx).Float() * 1024)
	cpu_formated, _ := df.Col("cpu").Elem(idx).Int()

	job := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: fmt.Sprintf("%dm", cpu_formated),
		MemRequested: fmt.Sprintf("%dMi", mem_formated),
		Label:        df.Col("label").Elem(idx).String(),
		Annotations: map[string]string{
			"pod-complete.stage.kwok.x-k8s.io/delay": df.Col("job_duration").Elem(idx).String() + "s",
		},
	}

	action := df.Col("action").Elem(idx).String()
	action = strings.ToLower(action)

	utils.Log_info(fmt.Sprintf("➡️ [%ss] [Job] %s: %s", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), job.Name))
	if action == "create" {
		err := events.Create_Workload(dynamicContext, job, "batch", "jobs")

		utils.Log_err("Failed to create Job", err)
	} else if action == "delete" {
		err := events.Job_delete(dynamicContext, job.Name)

		utils.Log_err("Failed to delete Job", err)
	} else if action == "update" {
		err := events.Job_update(dynamicContext, job)

		utils.Log_err("Failed to update Job", err)
	} else {
		utils.Log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf(""))
	}
}
