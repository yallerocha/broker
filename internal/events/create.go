package events

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	utils "github.com/cloud-ai-ufcg/broker/pkg/utils"
)

// Submit the workload using the dynamic client.
// Receives a dynamic client, a data struct representing
// the workload to submit, a resource to define the data type.
func Create_Workload(dynClient *dynamic.DynamicClient, data utils.Workload, group string, resource string) error {
	var workload any
	namespace := utils.Get_context_namespace()

	if err := validate_workload_parameter(group, resource); err != nil {
		return err
	}

	workload = create_manifest(resource, data)

	deployJSON, err := json.Marshal(workload)
	if err != nil {
		return err
	}

	unstructuredObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(deployJSON, unstructuredObj); err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  "v1",
		Resource: resource,
	}

	_, err = dynClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})

	return err
}

// Prepare a Deployment type for submission.
func deployment_create(data utils.Workload) *appv1.Deployment {
	label := map[string]string{}

	tolerations := []corev1.Toleration{{
		Key:      "kwok-provider",
		Operator: corev1.TolerationOpEqual,
		Value:    "true",
		Effect:   corev1.TaintEffectNoSchedule,
	}}

	if !slices.Contains([]string{"", "na", "n/a", "nan", "none"}, data.Label) {
		label["cloud"] = data.Label
	}

	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        data.Name,
			Labels:      label,
			Annotations: data.Annotations,
		},
		Spec: appv1.DeploymentSpec{
			Replicas: int32Ptr(data.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "deployment-app"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "deployment-app"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "busybox",
							Image:   "busybox:latest",
							Command: []string{"sh", "-c", "while true; do echo running; sleep 10; done"},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
									corev1.ResourceMemory: resource.MustParse(data.MemRequested),
								},
							},
						},
					},
					Tolerations: tolerations,
				},
			},
		},
	}

	deployment.Kind = "Deployment"
	deployment.APIVersion = "apps/v1"

	return deployment
}

// Prepare a Job type for submition.
func job_create(data utils.Workload) *batchv1.Job {
	label := map[string]string{}

	tolerations := []corev1.Toleration{{
		Key:      "kwok-provider",
		Operator: corev1.TolerationOpEqual,
		Value:    "true",
		Effect:   corev1.TaintEffectNoSchedule},
	}

	if !slices.Contains([]string{"", "na", "n/a", "nan", "none"}, data.Label) {
		label["cloud"] = data.Label
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        data.Name,
			Labels:      label,
			Annotations: data.Annotations,
		},
		Spec: batchv1.JobSpec{
			Completions: int32Ptr(data.Replicas),
			Parallelism: int32Ptr(data.Replicas),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"job": "job-app"},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "busybox",
							Image: "busybox:latest",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
									corev1.ResourceMemory: resource.MustParse(data.MemRequested),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
									corev1.ResourceMemory: resource.MustParse(data.MemRequested),
								},
							},
							Command: []string{"sh", "-c", "echo Hello World! && sleep 30"},
						},
					},
					Tolerations: tolerations,
				},
			},
		},
	}

	job.Kind = "Job"
	job.APIVersion = "batch/v1"

	return job
}

// Validate the parameters used in CreateWorkload function.
func validate_workload_parameter(group string, resource string) error {
	if !slices.Contains([]string{"batch", "apps"}, group) {
		return fmt.Errorf("invalid Group: %s", group)
	}

	if !slices.Contains([]string{"jobs", "deployments"}, resource) {
		return fmt.Errorf("invalid Resource: %s", resource)
	}

	return nil
}

// Use the other functions to create a manifest object for an especific resource type
// Receives the resource type (e.g. job, deployment, ...) and a workload object
// The object returned can be a deployment or a job.
func create_manifest(resource_type string, data utils.Workload) any {
	var out any
	resource_type = strings.ToLower(resource_type)

	if resource_type == "deployments" {
		out = deployment_create(data)
	} else if resource_type == "jobs" {
		out = job_create(data)
	}

	return out
}

func int32Ptr(i int32) *int32 {
	return &i
}
