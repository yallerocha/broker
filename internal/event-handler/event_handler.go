package eventhandler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cloud-ai-ufcg/broker/internal/events"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var wg sync.WaitGroup

func Handler(config utils.Config, origin_data dataframe.DataFrame) {
	clientset, err := utils.GetClientSet()
	utils.Exit_if_err(err, "Failed to get kubecontext")

	start_time := time.Now()
	df := origin_data.Arrange(dataframe.Sort("timestamp"))
	rows := df.Records()

	for i := range rows[0:df.Nrow()] {
		kind := df.Col("kind").Elem(i).String()
		time_stamp, err := df.Col("timestamp").Elem(i).Int()
		utils.Exit_if_err(err, "Failed to read the timestamp")

		sleep_time(time_stamp, start_time)
		wg.Add(1)

		if strings.ToLower(kind) == "deployment" {
			go deployment_action(clientset, df, i)
		} else if strings.ToLower(kind) == "job" {
			go job_action(clientset, df, i)
		} else {
			log.Fatalf("Unknown kind: %s", kind)
		}

	}

	wg.Wait()
}

func sleep_time(time_stamp int, start_time time.Time) {
	elapsed := time.Since(start_time)

	if elapsed.Seconds() < float64(time_stamp) {
		log.Printf("Waiting until %f seconds", float64(time_stamp))
		time.Sleep(time.Duration(float64(time_stamp)-elapsed.Seconds()) * time.Second)
	}

}

func deployment_action(clientset *kubernetes.Clientset, df dataframe.DataFrame, idx int) {
	defer wg.Done()

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

	if action == "create" {
		deployment_created := events.Deployment_create(deployment)
		_, err := clientset.AppsV1().Deployments("default").Create(context.Background(), deployment_created, metav1.CreateOptions{})

		utils.Exit_if_err(err, "Failed to create Deployment")
	} else if action == "delete" {
		events.Deployment_delete(clientset, deployment.Name, "default")
	} else if action == "update" {
		events.Deployment_update(clientset, deployment)
	} else {
		log.Fatalf("Unknown action: %s", action)
	}

}

func job_action(clientset *kubernetes.Clientset, df dataframe.DataFrame, idx int) {
	defer wg.Done()

	replicas, _ := df.Col("replicas").Elem(idx).Int()
	mem_formated := int64(df.Col("memory").Elem(idx).Float() * 1024)

	job := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: df.Col("cpu").Elem(idx).String(),
		MemRequested: fmt.Sprintf("%dMi", mem_formated),
		Label:        df.Col("label").Elem(idx).String(),
		Annotations: map[string]string{
			"pod-complete.stage.kwok.x-k8s.io/delay": df.Col("job_duration").Elem(idx).String() + "s",
		},
	}

	action := df.Col("action").Elem(idx).String()
	action = strings.ToLower(action)

	if action == "create" {
		job_created := events.Job_create(job)
		_, err := clientset.BatchV1().Jobs("default").Create(context.Background(), job_created, metav1.CreateOptions{})

		utils.Exit_if_err(err, "Failed to create Job")
	} else if action == "delete" {
		events.Job_delete(clientset, job.Name, "default")
	} else if action == "update" {
		events.Job_update(clientset, job)
	} else {
		log.Fatalf("Unknown action: %s", action)
	}
}
