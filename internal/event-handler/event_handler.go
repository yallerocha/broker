package eventhandler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/cloud-ai-ufcg/broker/internal/events"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var wg sync.WaitGroup

func Handler(config utils.Config, df dataframe.DataFrame) {
	clientset, err := utils.GetClientSet()
	utils.Exit_if_err(err, "Failed to get kubecontext")
	rows := df.Records()

	for i := range rows[0:df.Nrow()] {
		kind := df.Col("kind").Elem(i).String()
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

func deployment_action(clientset *kubernetes.Clientset, df dataframe.DataFrame, idx int) {
	fmt.Println("asdasda")

	defer wg.Done()
	replicas, _ := df.Col("replicas").Elem(idx).Int()
	mem_format := "Mi"

	deployment := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: df.Col("cpu").Elem(idx).String(),
		MemRequested: df.Col("memory").Elem(idx).String() + mem_format,
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

	} else {
		log.Fatalf("Unknown action: %s", action)
	}

}

func job_action(clientset *kubernetes.Clientset, df dataframe.DataFrame, idx int) {
	defer wg.Done()
	replicas, _ := df.Col("replicas").Elem(idx).Int()

	job := utils.Workload{
		Name:         df.Col("id").Elem(idx).String(),
		Replicas:     int32(replicas),
		CpuRequested: df.Col("cpu").Elem(idx).String(),
		MemRequested: df.Col("memory").Elem(idx).String(),
		Label:        df.Col("label").Elem(idx).String(),
		Annotations: map[string]string{
			"pod-complete.stage.kwok.x-k8s.io/delay": df.Col("job").Elem(idx).String() + "s",
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

	} else {
		log.Fatalf("Unknown action: %s", action)
	}
}
