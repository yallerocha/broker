package events

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Update the deployment using a clientset.
// Receives a logger, the data representing a workload.
func Deployment_update(logger *slog.Logger, clientset *kubernetes.Clientset, data utils.Workload) *v1.Deployment {
	namespace := "default"

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), data.Name, meta.GetOptions{})
	exit_if_err(logger, err, "Failed to get deployment")

	// resource update in each container

	for i := range deployment.Spec.Template.Spec.Containers {
		deployment.Spec.Template.Spec.Containers[i].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
				corev1.ResourceMemory: resource.MustParse(data.MemRequested),
			},
		}
	}

	deployment.Spec.Replicas = int32Ptr(data.Replicas)
	label := strings.TrimSpace(strings.ToLower(data.Label))

	// label and annotation update

	if deployment.ObjectMeta.Labels == nil {
		deployment.ObjectMeta.Labels = map[string]string{}
	}

	if deployment.ObjectMeta.Annotations == nil {
		deployment.ObjectMeta.Annotations = map[string]string{}
	}

	if slices.Contains([]string{"", "na", "n/a", "nan", "none"}, label) {
		delete(deployment.ObjectMeta.Labels, "cloud")
		delete(deployment.ObjectMeta.Annotations, "clusterpropagationpolicy.karmada.io/name")
	} else {
		deployment.ObjectMeta.Labels["cloud"] = label
		deployment.ObjectMeta.Annotations["clusterpropagationpolicy.karmada.io/name"] = "deploy-" + label
	}

	// apply the update

	update, err := clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, meta.UpdateOptions{})
	exit_if_err(logger, err, "Failed to update deployment")

	return update
}

// Update the Job using a dynamicContext.
// Receives a logger, the data representing a workload.
// If has any difference the job will be recreated.
func Job_update(logger *slog.Logger, clientset *kubernetes.Clientset, dynamicContext *dynamic.DynamicClient, data utils.Workload) {
	namespace := "default"

	job, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), data.Name, meta.GetOptions{})
	exit_if_err(logger, err, "Failed to get job")

	// recreate if has difference

	container_resource := job.Spec.Template.Spec.Containers[0]
	currentCpu := container_resource.Resources.Requests.Cpu().MilliValue()
	currentMem := container_resource.Resources.Requests.Memory().Value() / (1024 * 1024)

	newCpu := resource.MustParse(data.CpuRequested)
	newMem := strings.Split(data.MemRequested, "Mi")
	newMemConverted, err := strconv.ParseFloat(newMem[0], 64)

	exit_if_err(logger, err, "Failed during convert process")

	hasDiff := currentCpu != newCpu.MilliValue() || math.Abs(newMemConverted-float64(currentMem)) > 0.0001 || *job.Spec.Completions != data.Replicas

	if hasDiff {
		Job_delete(logger, clientset, data.Name, namespace)

		time.Sleep(500 * time.Millisecond)
		err := Create_Workload(dynamicContext, data, "batch", "jobs")

		exit_if_err(logger, err, "Failed to update job")
		return
	}

	// update the annotations and labels

	label := strings.TrimSpace(strings.ToLower(data.Label))
	empty := []string{"", "na", "n/a", "nan", "none"}

	if job.ObjectMeta.Labels == nil {
		job.ObjectMeta.Labels = map[string]string{}
	}

	if job.ObjectMeta.Annotations == nil {
		job.ObjectMeta.Annotations = map[string]string{}
	}

	if slices.Contains(empty, label) {
		delete(job.ObjectMeta.Labels, "cloud")
		delete(job.ObjectMeta.Annotations, "clusterpropagationpolicy.karmada.io/name")
	} else {
		job.ObjectMeta.Labels["cloud"] = label
		job.ObjectMeta.Annotations["clusterpropagationpolicy.karmada.io/name"] = "job-" + label
	}

	if len(data.Annotations) > 0 {
		job.ObjectMeta.Annotations["pod-complete.stage.kwok.x-k8s.io/delay"] = data.Annotations["pod-complete.stage.kwok.x-k8s.io/delay"]
	}

	// Perform the update

	_, err = clientset.BatchV1().Jobs(namespace).Update(context.TODO(), job, meta.UpdateOptions{})
	exit_if_err(logger, err, "Failed to update job")
}

func exit_if_err(logger *slog.Logger, err error, msg string) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "???"
			line = 0
		}

		logger.Error("❌ "+msg+": "+err.Error(), slog.String("source", fmt.Sprintf("%s:%d", file, line)))
	}
}
