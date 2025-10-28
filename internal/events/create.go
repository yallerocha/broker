package events

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

// Create_Workload submits a Kubernetes workload (Deployment, Job, or Node) using a dynamic client.
// It receives a dynamic client connected to the target cluster, a Workload data struct,
// the API group (e.g., "apps", "batch", "core"), and the resource type (e.g., "deployments", "jobs", "nodes").
func Create_Workload(dynClient *dynamic.DynamicClient, data utils.Workload, group string, resource string) error {
	var workload any
	namespace := utils.Get_context_namespace() // Namespace is relevant for namespaced resources

	// Validate the provided API group and resource type.
	if err := validate_workload_parameter(group, resource); err != nil {
		return err
	}

	// Create the specific Kubernetes manifest object (Deployment, Job, or Node).
	workload = create_manifest(resource, data)

	// Marshal the Go struct into JSON for Unstructured conversion.
	deployJSON, err := json.Marshal(workload)
	if err != nil {
		return err
	}

	unstructuredObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(deployJSON, unstructuredObj); err != nil {
		return err
	}

	targetGroup := group
	// For core API group resources (like Nodes), the Group field should be empty in GVR.
	if group == "core" {
		targetGroup = ""
	}

	gvr := schema.GroupVersionResource{
		Group:    targetGroup,
		Version:  "v1",
		Resource: resource,
	}

	// Create the resource: Nodes are cluster-scoped, others are namespaced.
	if targetGroup == "" && resource == "nodes" {
		_, err = dynClient.Resource(gvr).Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	} else {
		_, err = dynClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	}

	return err
}

// deployment_create prepares a Kubernetes Deployment object for submission.
// It populates the Deployment's metadata, spec (replicas, image, commands, resource requests),
// and tolerations based on the provided Workload data.
func deployment_create(data utils.Workload) *appv1.Deployment {
	label := map[string]string{}

	// Only add tolerations if running in KWOK mode
	// In real mode, nodes don't have kwok-provider taint, so tolerations would prevent scheduling
	tolerations := []corev1.Toleration{}

	// Check if we're in KWOK mode by environment variable or default to false
	// If KWOK_MODE is set to "true", add tolerations
	if strings.ToLower(os.Getenv("KWOK_MODE")) == "true" {
		tolerations = []corev1.Toleration{{
			Key:      "kwok-provider",
			Operator: corev1.TolerationOpEqual,
			Value:    "true",
			Effect:   corev1.TaintEffectNoSchedule,
		}}
	}

	if !slices.Contains([]string{"", "na", "n/a", "nan", "none"}, strings.ToLower(data.Label)) {
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
							Name:    "cpu-worker",
							Image:   "containerstack/cpustress:latest",
							Command: []string{"sh", "-c", "stress-ng --cpu $(nproc) --timeout 0s"},
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
						},
					},
					Tolerations: tolerations, // Conditionally apply Kwok tolerations
				},
			},
		},
	}

	deployment.Kind = "Deployment"
	deployment.APIVersion = "apps/v1"

	return deployment
}

// job_create prepares a Kubernetes Job object for submission.
// It populates the Job's metadata, spec (completions, parallelism, image, commands, resource requests/limits),
// and tolerations based on the provided Workload data.
func job_create(data utils.Workload) *batchv1.Job {
	label := map[string]string{}

	// Only add tolerations if running in KWOK mode
	// In real mode, nodes don't have kwok-provider taint, so tolerations would prevent scheduling
	tolerations := []corev1.Toleration{}

	// Check if we're in KWOK mode by environment variable
	if strings.ToLower(os.Getenv("KWOK_MODE")) == "true" {
		tolerations = []corev1.Toleration{{
			Key:      "kwok-provider",
			Operator: corev1.TolerationOpEqual,
			Value:    "true",
			Effect:   corev1.TaintEffectNoSchedule,
		}}
	}

	// Apply a custom "cloud" label if specified in the workload data.
	if !slices.Contains([]string{"", "na", "n/a", "nan", "none"}, strings.ToLower(data.Label)) {
		label["cloud"] = data.Label
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   data.Name,
			Labels: label,
		},
		Spec: batchv1.JobSpec{
			Completions: int32Ptr(data.Replicas),
			Parallelism: int32Ptr(data.Replicas), // Run replicas in parallel
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"job": "job-app"}, // Selector for the Pod Template
					Annotations: data.Annotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever, // Jobs are typically not restarted on completion
					Containers: []corev1.Container{
						{
							Name:    "cpu-worker-job",
							Image:   "containerstack/cpustress:latest",
							Command: []string{"sh", "-c", "stress-ng --cpu 1 --timeout 30s && echo Job completed"},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
									corev1.ResourceMemory: resource.MustParse(data.MemRequested),
								},
								Limits: corev1.ResourceList{ // Often limits are set to requests for Jobs
									corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
									corev1.ResourceMemory: resource.MustParse(data.MemRequested),
								},
							},
						},
					},
					Tolerations: tolerations, // Apply Kwok tolerations
				},
			},
		},
	}

	job.Kind = "Job"
	job.APIVersion = "batch/v1"

	return job
}

// node_create prepares a Kubernetes Node object (specifically for Kwok simulated nodes) for submission.
// It populates the Node's metadata, spec (taints), and status (allocatable/capacity resources, node info)
// based on the provided Workload data.
func node_create(data utils.Workload) *corev1.Node {
	label := map[string]string{}
	annotations := map[string]string{
		"node.alpha.kubernetes.io/ttl": "0",    // Kwok specific TTL
		"kwok.x-k8s.io/node":           "fake", // Identifies as a fake Kwok node
	}

	// Apply a custom "cloud" label if specified in the workload data.
	if !slices.Contains([]string{"", "na", "n/a", "nan", "none"}, strings.ToLower(data.Label)) {
		label["cloud"] = data.Label
	}

	// Create the Node object based on the provided YAML manifesto structure.
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: data.Name,
			Labels: map[string]string{
				"beta.kubernetes.io/arch":       "amd64",
				"beta.kubernetes.io/os":         "linux",
				"kubernetes.io/arch":            "amd64",
				"kubernetes.io/hostname":        data.Name,
				"kubernetes.io/os":              "linux",
				"kubernetes.io/role":            "agent",
				"node-role.kubernetes.io/agent": "",
				"type":                          "kwok",
				"cloud":                         data.Label, // Dynamic cloud label
			},
			Annotations: annotations,
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{ // Apply Kwok provider taint to prevent regular scheduling
				{
					Effect: corev1.TaintEffectNoSchedule,
					Key:    "kwok-provider",
					Value:  "true",
				},
			},
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
				corev1.ResourceMemory: resource.MustParse(data.MemRequested),
				corev1.ResourcePods:   resource.MustParse("110"),
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
				corev1.ResourceMemory: resource.MustParse(data.MemRequested),
				corev1.ResourcePods:   resource.MustParse("110"),
			},
			Phase: corev1.NodeRunning, // Set node status to "Running"
			NodeInfo: corev1.NodeSystemInfo{
				Architecture:            "amd64",
				BootID:                  "",
				ContainerRuntimeVersion: "",
				KernelVersion:           "",
				KubeProxyVersion:        "fake",
				KubeletVersion:          "fake",
				MachineID:               "",
				OperatingSystem:         "linux",
				OSImage:                 "",
				SystemUUID:              "",
			},
		},
	}

	node.Kind = "Node"
	node.APIVersion = "v1"

	return node
}

// validate_workload_parameter validates the provided API group and resource type
// to ensure they are supported for creation.
func validate_workload_parameter(group string, resource string) error {
	if !slices.Contains([]string{"batch", "apps", "core"}, group) {
		return fmt.Errorf("invalid Group: %s", group)
	}

	if !slices.Contains([]string{"jobs", "deployments", "nodes"}, resource) {
		return fmt.Errorf("invalid Resource: %s", resource)
	}

	return nil
}

// create_manifest acts as a factory function to create a Kubernetes manifest object
// (Deployment, Job, or Node) based on the specified resource type and Workload data.
func create_manifest(resource_type string, data utils.Workload) any {
	var out any
	resource_type = strings.ToLower(resource_type) // Ensure case-insensitivity

	if resource_type == "deployments" {
		out = deployment_create(data)
	} else if resource_type == "jobs" {
		out = job_create(data)
	} else if resource_type == "nodes" {
		out = node_create(data)
	} else {
		// Log an error if an unknown resource type is requested.
		utils.Log_err(fmt.Sprintf("Unknown resource type: %s", resource_type), nil)
		out = nil
	}

	return out
}

// int32Ptr is a helper function to return a pointer to an int32 value.
// It's commonly used when Kubernetes API fields require a pointer to an int32 (e.g., Replicas).
func int32Ptr(i int32) *int32 {
	return &i
}
