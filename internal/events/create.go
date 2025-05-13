package events

import (
	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utils "github.com/cloud-ai-ufcg/broker/pkg/utils"
)

func Deployment_create(data utils.Workload) *appv1.Deployment {

	tolerations := []corev1.Toleration{{
		Key:      "kwok-provider",
		Operator: corev1.TolerationOpEqual,
		Value:    "true",
		Effect:   corev1.TaintEffectNoSchedule,
	}}

	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: data.Name,
		},
		Spec: appv1.DeploymentSpec{
			Replicas: int32Ptr(data.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "deployment-app"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "deployment-app"},
					Annotations: data.Annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "busybox",
							Image:   "busybox:latest",
							Command: []string{"sh", "-c", "echo Hello World! && sleep 30"},
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

	return deployment
}

func Job_create(data utils.Workload, job_duration string) *batchv1.Job {

	tolerations := []corev1.Toleration{{
		Key:      "kwok-provider",
		Operator: corev1.TolerationOpEqual,
		Value:    "true",
		Effect:   corev1.TaintEffectNoSchedule},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: data.Name,
		},
		Spec: batchv1.JobSpec{
			Completions: int32Ptr(data.Replicas),
			Parallelism: int32Ptr(data.Replicas),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: data.Annotations,
					Labels:      map[string]string{"job": "job-app"},
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

	return job
}

func int32Ptr(i int32) *int32 {
	return &i
}
