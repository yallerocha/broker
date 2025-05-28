package eventhandler

import (
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/cloud-ai-ufcg/broker/internal/events"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	logger *slog.Logger
)

// Entrypoint to run the event handler.
// It receives a config 'struct' and a dataframe containing the data
func Handler(config utils.Config, origin_data dataframe.DataFrame, default_logger *slog.Logger) {
	clientset, err := utils.GetClientSet(config.KubeConfig)
	dynamicContext := utils.GetDynamicContext(default_logger)

	logger = default_logger
	log_err("Failed to get the client context", err)

	start_time := time.Now()
	df := origin_data.Arrange(dataframe.Sort("timestamp"))
	rows := df.Records()

	for i := range rows[0:df.Nrow()] {
		kind := df.Col("kind").Elem(i).String()
		time_stamp, err := df.Col("timestamp").Elem(i).Int()
		log_err("Failed to read the timestamp", err)

		sleep_time(time_stamp, start_time)

		if strings.ToLower(kind) == "deployment" {
			deployment_action(clientset, dynamicContext, df, i)
		} else if strings.ToLower(kind) == "job" {
			job_action(clientset, dynamicContext, df, i)
		} else {
			log_err(fmt.Sprintf("Unknown kind %s", kind), fmt.Errorf(""))
		}

	}

}

// Apply the sleep time to an execution time less then the 'time stamp'
// Receives the current time stamp in the dataframe and the start time.
func sleep_time(time_stamp int, start_time time.Time) {
	elapsed := time.Since(start_time)

	if int64(math.Ceil(elapsed.Seconds())) < int64(time_stamp) {
		logger.Info(fmt.Sprintf("⏳ Waiting %d seconds", int64(time_stamp)))
		time.Sleep(time.Duration(float64(time_stamp)-elapsed.Seconds()) * time.Second)
	}

}

// Execute the action required for each deployment
// Receives the clientset and dynamicContext for requests,
// a df containing the data and an idx that represents the index of this workload.
func deployment_action(clientset *kubernetes.Clientset, dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
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

	logger.Info(fmt.Sprintf("➡️ [%ss] [Deployment] %s: %s", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), deployment.Name))
	if action == "create" {
		err := events.Create_Workload(dynamicContext, deployment, "apps", "deployments")

		log_err("Failed to create Deployment", err)
	} else if action == "delete" {
		events.Deployment_delete(logger, dynamicContext, deployment.Name, "default")
	} else if action == "update" {
		events.Deployment_update(logger, clientset, deployment)
	} else {
		log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf(""))
	}

}

// Execute the action required for each deployment
// Receives the clientset and dynamicContext for requests,
// a df containing the data and an idx that represents the index of this workload.
func job_action(clientset *kubernetes.Clientset, dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
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

	logger.Info(fmt.Sprintf("➡️ [%ss] [Job] %s: %s", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), job.Name))
	if action == "create" {
		err := events.Create_Workload(dynamicContext, job, "batch", "jobs")

		log_err("Failed to create Job", err)
	} else if action == "delete" {
		events.Job_delete(logger, dynamicContext, job.Name, "default")
	} else if action == "update" {
		events.Job_update(logger, clientset, dynamicContext, job)
	} else {
		log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf(""))
	}
}

// Define a function to loggers.
func log_err(msg string, err error) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "???"
			line = 0
		}

		logger.Error("❌ "+msg+": "+err.Error(), slog.String("source", fmt.Sprintf("%s:%d", file, line)))
	}
}
