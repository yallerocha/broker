package events

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Update the deployment using a clientset.
// Receives the data representing a workload.
func Deployment_update(dynamicContext *dynamic.DynamicClient, data utils.Workload) error {
	namespace := "default"

	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	// get deployment as Unstructured
	unstr, err := dynamicContext.Resource(gvr).Namespace(namespace).Get(context.TODO(), data.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// convert Unstructured to typed Deployment
	var deployment v1.Deployment
	err = apiruntime.DefaultUnstructuredConverter.FromUnstructured(unstr.Object, &deployment)
	if err != nil {
		return err
	}

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

	// convert to Unstructured
	objMap, err := apiruntime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
	if err != nil {
		return err
	}
	unstr.Object = objMap

	// apply update
	_, err = dynamicContext.Resource(gvr).Namespace(namespace).Update(context.TODO(), unstr, metav1.UpdateOptions{})

	if err != nil {
		return err
	}

	return nil
}

// Update the Job using a dynamicContext.
// Receives a logger, the data representing a workload.
// If has any difference the job will be recreated.
func Job_update(dynamicContext *dynamic.DynamicClient, data utils.Workload) error {
	namespace := "default"
	deletion_timeout := 5

	gvr := schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}

	// get Job as Unstructured
	unstr, err := dynamicContext.Resource(gvr).Namespace(namespace).Get(context.TODO(), data.Name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	// convert to batchv1.Job
	var job batchv1.Job
	err = apiruntime.DefaultUnstructuredConverter.FromUnstructured(unstr.Object, &job)

	if err != nil {
		return err
	}

	// recreate if has difference

	container_resource := job.Spec.Template.Spec.Containers[0]
	currentCpu := container_resource.Resources.Requests.Cpu().MilliValue()
	currentMem := container_resource.Resources.Requests.Memory().Value() / (1024 * 1024)

	newCpu := resource.MustParse(data.CpuRequested)
	newMem := strings.Split(data.MemRequested, "Mi")
	newMemConverted, err := strconv.ParseFloat(newMem[0], 64)

	if err != nil {
		return err
	}

	hasDiff := currentCpu != newCpu.MilliValue() || math.Abs(newMemConverted-float64(currentMem)) > 0.0001 || *job.Spec.Completions != data.Replicas

	if hasDiff {

		if err := Job_delete(dynamicContext, data.Name, namespace); err != nil {
			return err
		}

		if err := waitForJobDeletion(dynamicContext, data.Name, namespace, time.Duration(deletion_timeout)); err != nil {
			return err
		}

		return Create_Workload(dynamicContext, data, "batch", "jobs")
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

	// perform the update

	objMap, err := apiruntime.DefaultUnstructuredConverter.ToUnstructured(&job)

	if err != nil {
		return err
	}

	unstr.Object = objMap

	// apply update using dynamic client
	_, err = dynamicContext.Resource(gvr).Namespace(namespace).Update(context.TODO(), unstr, metav1.UpdateOptions{})

	return err
}

// Wait for total job deletion.
// Receives a dynamic context, the name of the job, the namespace and the timeout for limiting this wait.
func waitForJobDeletion(dynamicClient dynamic.Interface, name string, namespace string, timeout time.Duration) error {
	gvr := schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	start := time.Now()

	for {
		_, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})

		// if the job is already deleted
		if meta.IsNoMatchError(err) || k8serrors.IsNotFound(err) {
			return nil
		}

		// if another error appears during execution
		if err != nil && !strings.Contains(err.Error(), "being deleted") {
			return fmt.Errorf("unexpected error while waiting for deletion: %w", err)
		}

		// timeout
		if time.Duration(time.Since(start).Seconds()) > time.Duration(timeout.Seconds()) {
			return fmt.Errorf("timeout (%d s) while waiting for job %s deletion", int(timeout.Seconds()), name)
		}

		time.Sleep(200 * time.Millisecond)
	}
}
