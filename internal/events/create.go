package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
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
	"github.com/gomorpheus/morpheus-go-sdk"
	sigsyaml "sigs.k8s.io/yaml"
)

// Workload container images. Estas imagens são multi-arquitetura (incluem
// linux/ppc64le, amd64, arm64, s390x), então funcionam tanto em x86 quanto em
// Power9. As imagens antigas (polinux/stress-ng, containerstack/cpustress) só
// possuíam build amd64, fazendo os pods ficarem em ImagePullBackOff em ppc64le.
const (
	// stressNgImage é baseada em debian:12 e tem stress-ng e /bin/sh no PATH,
	// servindo tanto para comandos diretos ("stress-ng ...") quanto "sh -c".
	stressNgImage = "ghcr.io/colinianking/stress-ng:latest"
	// idleImage e jobImage usam stress-ng/busybox multi-arch.
	idleImage = "busybox:latest"
	jobImage  = "ghcr.io/colinianking/stress-ng:latest"
)

// Morpheus_Create_Workload submits a Kubernetes workload (Deployment, Job, or Node) using a morpheus client.
// It receives a morpheus client connected to the target cluster and a Workload data struct
func Morpheus_Create_Workload(morpheusClient *utils.MorpheusClient, workload utils.Workload) error {
	yamlStr := create_yaml_template(workload)
	if yamlStr == "" {
		return fmt.Errorf("failed to generate YAML for workload %s", workload.Name)
	}
	req := &morpheus.Request{
		Body: map[string]interface{}{"specYaml": yamlStr},
	}

	// workload.Label is expected to carry the target Morpheus cluster ID as a string.
	clusterID, err := strconv.ParseInt(workload.Label, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid cluster id in workload.Label: %s", workload.Label)
	}

	_, err = morpheusClient.Client.ApplyTemplateToCluster(clusterID, req)
	if err != nil {
		return fmt.Errorf("failed to apply deployment YAML to cluster %d: %v", clusterID, err)
	}
	return nil
}

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

func create_yaml_template(workload utils.Workload) string {
	dep := deployment_create(workload)
	jsonBytes, err := json.Marshal(dep)
	if err != nil {
		return ""
	}
	yamlBytes, err := sigsyaml.JSONToYAML(jsonBytes)
	if err != nil {
		return ""
	}
	return string(yamlBytes)
}

// This ensures high memory utilization while leaving overhead for the container itself
func calculateMemoryBytes(memRequested string) string {
	// Parse the memory request (e.g., "2048Mi", "2Gi", "1024Mi")
	quantity, err := resource.ParseQuantity(memRequested)
	if err != nil {
		log.Printf("⚠️  Failed to parse memory request '%s', defaulting to 900M", memRequested)
		return "900M"
	}

	// Get value in bytes
	memBytes := quantity.Value()

	// Use 85% of available memory to leave overhead for stress-ng binary, stack, and OS
	// This prevents OOMKilled errors when stress-ng tries to allocate memory
	targetBytes := int64(float64(memBytes) * 0.85)

	// Convert to megabytes for stress-ng (which expects M suffix)
	targetMB := targetBytes / (1024 * 1024)

	// Ensure at least 1MB is allocated
	if targetMB < 1 {
		targetMB = 1
	}

	result := fmt.Sprintf("%dM", targetMB)
	log.Printf("💾 Memory-intensive workload: requested=%s, stress-ng will use=%s (85%% to prevent OOM)", memRequested, result)
	return result
}

// getWorkloadContainer returns the appropriate container configuration based on workload type
func getWorkloadContainer(workloadType string, data utils.Workload) corev1.Container {
	// Check if running in KWOK simulation mode
	kwokMode := os.Getenv("KWOK_MODE")
	if kwokMode == "true" {
		log.Println("🎭 KWOK Mode: Creating fake container (busybox)")
		return corev1.Container{
			Name:    "workload",
			Image:   idleImage,
			Command: []string{"sh", "-c", "echo 'Simulated workload' && sleep 3600"},
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
		}
	}

	// Real mode: Parse resource requirements
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
			corev1.ResourceMemory: resource.MustParse(data.MemRequested),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(data.CpuRequested),
			corev1.ResourceMemory: resource.MustParse(data.MemRequested),
		},
	}

	// Normalize workload type
	wType := strings.ToLower(strings.TrimSpace(workloadType))

	switch wType {
	case "memory-intensive":
		memBytes := calculateMemoryBytes(data.MemRequested)
		return corev1.Container{
			Name:      "memory-worker",
			Image:     stressNgImage,
			Command:   []string{"stress-ng", "--vm", "1", "--vm-bytes", memBytes, "--vm-keep", "--timeout", "0s"},
			Resources: resources,
		}

	case "io-intensive":
		return corev1.Container{
			Name:      "io-worker",
			Image:     stressNgImage,
			Command:   []string{"stress-ng", "--io", "4", "--hdd", "2", "--timeout", "0s"},
			Resources: resources,
		}

	case "network-intensive":
		return corev1.Container{
			Name:      "network-worker",
			Image:     stressNgImage,
			Command:   []string{"stress-ng", "--sock", "4", "--timeout", "0s"},
			Resources: resources,
		}

	case "mixed":
		memBytes := calculateMemoryBytes(data.MemRequested)
		return corev1.Container{
			Name:      "mixed-worker",
			Image:     stressNgImage,
			Command:   []string{"stress-ng", "--cpu", "1", "--vm", "1", "--vm-bytes", memBytes, "--vm-keep", "--io", "1", "--timeout", "0s"},
			Resources: resources,
		}

	case "bursty":
		return corev1.Container{
			Name:      "bursty-worker",
			Image:     stressNgImage,
			Command:   []string{"sh", "-c", "while true; do stress-ng --cpu 2 --timeout 10s; sleep 20; done"},
			Resources: resources,
		}

	case "realistic":
		// Simulates a real web application with variable load patterns:
		// - Base CPU load (~30% of requested)
		// - Memory working set that grows/shrinks
		// - Periodic spikes simulating traffic bursts
		// - Network activity
		memBytes := calculateMemoryBytes(data.MemRequested)
		return corev1.Container{
			Name:  "realistic-worker",
			Image: stressNgImage,
			Command: []string{"sh", "-c", `
				while true; do
					# Base load phase (30 seconds) - simulates normal traffic
					stress-ng --cpu 1 --cpu-load 30 --vm 1 --vm-bytes ` + memBytes + ` --sock 2 --timeout 30s;
					
					# Medium load phase (20 seconds) - simulates increased traffic
					stress-ng --cpu 1 --cpu-load 60 --vm 1 --vm-bytes ` + memBytes + ` --sock 4 --timeout 20s;
					
					# High load spike (10 seconds) - simulates traffic spike
					stress-ng --cpu 2 --cpu-load 90 --vm 1 --vm-bytes ` + memBytes + ` --vm-keep --sock 6 --io 2 --timeout 10s;
					
					# Cool down (20 seconds) - simulates low traffic period
					stress-ng --cpu 1 --cpu-load 20 --timeout 20s;
				done
			`},
			Resources: resources,
		}

	case "microservice":
		// Simulates a typical microservice with network-heavy, low CPU/memory usage
		// Good for testing network-based decisions
		return corev1.Container{
			Name:  "microservice-worker",
			Image: stressNgImage,
			Command: []string{"sh", "-c", `
				while true; do
					# Normal operation - mostly network I/O with occasional CPU spikes
					stress-ng --sock 4 --cpu 1 --cpu-load 20 --timeout 45s;
					# Request processing spike
					stress-ng --sock 8 --cpu 1 --cpu-load 70 --timeout 15s;
				done
			`},
			Resources: resources,
		}

	case "database":
		// Simulates a database workload with I/O heavy operations and memory caching
		memBytes := calculateMemoryBytes(data.MemRequested)
		return corev1.Container{
			Name:  "database-worker",
			Image: stressNgImage,
			Command: []string{"sh", "-c", `
				while true; do
					# Normal query processing - I/O and memory heavy
					stress-ng --hdd 2 --io 4 --vm 1 --vm-bytes ` + memBytes + ` --vm-keep --timeout 40s;
					# Heavy query/backup simulation
					stress-ng --hdd 4 --io 8 --vm 1 --vm-bytes ` + memBytes + ` --vm-keep --cpu 1 --timeout 20s;
				done
			`},
			Resources: resources,
		}

	case "batch":
		// Simulates batch processing jobs with high resource usage followed by idle
		memBytes := calculateMemoryBytes(data.MemRequested)
		return corev1.Container{
			Name:  "batch-worker",
			Image: stressNgImage,
			Command: []string{"sh", "-c", `
				while true; do
					# Heavy processing phase
					stress-ng --cpu 2 --vm 1 --vm-bytes ` + memBytes + ` --vm-keep --io 2 --timeout 60s;
					# Idle phase between batches
					sleep 30;
				done
			`},
			Resources: resources,
		}

	case "idle":
		return corev1.Container{
			Name:      "idle-worker",
			Image:     idleImage,
			Command:   []string{"sh", "-c", "while true; do echo 'healthy'; sleep 30; done"},
			Resources: resources,
		}

	case "cpu-intensive":
		fallthrough
	default:
		// Default to CPU-intensive workload
		return corev1.Container{
			Name:      "cpu-worker",
			Image:     stressNgImage,
			Command:   []string{"stress-ng", "--cpu", "1", "--timeout", "0s"},
			Resources: resources,
		}
	}
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
						getWorkloadContainer(data.WorkloadType, data),
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
							Image:   jobImage,
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

	switch resource_type {
	case "deployments":
		out = deployment_create(data)
	case "jobs":
		out = job_create(data)
	case "nodes":
		out = node_create(data)
	default:
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
