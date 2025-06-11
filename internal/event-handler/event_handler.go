package eventhandler

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/cloud-ai-ufcg/broker/internal/events"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"k8s.io/client-go/dynamic"
)

// Entrypoint to run the event handler.
// It receives a dataframe containing the data
func Handler(origin_data dataframe.DataFrame) {
	dynamicContext, err := utils.GetDynamicContext()

	utils.Log_fatal("Failed to get the client context", err)

	start_time := time.Now()
	df := origin_data.Arrange(dataframe.Sort("timestamp"))
	rows := df.Records()

	for i := range rows[0:df.Nrow()] {
		kind := df.Col("kind").Elem(i).String()
		time_stamp, err := df.Col("timestamp").Elem(i).Int()
		utils.Log_err("Failed to read the timestamp", err)

		sleep_time(time_stamp, start_time)

		if strings.ToLower(kind) == "deployment" {
			deployment_action(dynamicContext, df, i)
		} else if strings.ToLower(kind) == "job" {
			job_action(dynamicContext, df, i)
		} else {
			utils.Log_err(fmt.Sprintf("Unknown kind %s", kind), fmt.Errorf("dataframe line %d", i+2))
		}

	}

}

// This function is responsible for synchronizing the broker execution time
// with the 'time_stamp' defined in the CSV. If the execution time is less than
// the 'time_stamp', it will apply a sleep using the difference between
// the execution time and the 'time_stamp'.
// Receives the current 'time_stamp' in the dataframe and the 'start_time'.
func sleep_time(time_stamp int, start_time time.Time) {
	elapsed := time.Since(start_time)

	if int64(math.Ceil(elapsed.Seconds())) < int64(time_stamp) {
		time_wait := float64(time_stamp) - elapsed.Seconds()

		utils.Log_info(fmt.Sprintf("⏳ Waiting %d seconds", int64(time_wait)))
		time.Sleep(time.Duration(time_wait) * time.Second)
	}

}

// Execute the action required for each deployment
// Receives the dynamicContext for requests,
// a df containing the data and an idx that represents the index of this workload.
func deployment_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
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
func job_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
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
