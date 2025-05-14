package events

import (
	"context"
	"log"
	"slices"
	"strings"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Deployment_update(clientset *kubernetes.Clientset, data utils.Workload) *v1.Deployment {
	namespace := "default"

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), data.Name, meta.GetOptions{})
	exit_if_nil(err, "Faile to get deployment")

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
	exit_if_nil(err, "Failed to update deployment")

	return update
}

func Job_update(data utils.Workload) {

}

func exit_if_nil(err error, msg string) {
	if err != nil {
		log.Fatalln(msg)
	}
}
