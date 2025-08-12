package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubepath  string
	namespace string
)

// Set the kubernetes context path. Este continua sendo usado para configurar o caminho do kubeconfig principal (do Karmada).
// Ele é chamado na inicialização do broker (geralmente em main.go).
func Set_context_path(path string) {
	kubepath = filepath.Join(
		os.Getenv("HOME"), ".kube", path, // Assume que karmada.config está em ~/.kube/
	)
	Log_info(fmt.Sprintf("Kubeconfig path for default context (Karmada) set to: %s", kubepath))
}

// Set the target namespace
func Set_context_namespace(namespace_config string) {
	namespace = namespace_config
}

// Get the target namespace
func Get_context_namespace() string {
	return namespace
}

// GetDynamicContext retorna um DynamicClient para o contexto principal do broker (o Karmada).
// Esta é a função original, que workloads usarão.
func GetDynamicContext() (*dynamic.DynamicClient, error) {
	if kubepath == "" {
		// Fallback ou erro se o kubeconfig do Karmada não for configurado.
		return nil, fmt.Errorf("karmada kubeconfig path not set. Call Set_context_path first.")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubepath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from flags for Karmada (%s): %w", kubepath, err)
	}

	config.QPS = 100
	config.Burst = 200

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client for Karmada (%s): %w", kubepath, err)
	}
	return dynClient, nil
}

// GetDynamicClientForMemberCluster retorna um DynamicClient para um cluster membro específico.
// Ele usa o 'members.config' e seleciona o contexto apropriado dentro dele.
// Esta é a nova função, usada especificamente para Nodes.
func GetDynamicClientForMemberCluster(memberLabel string) (*dynamic.DynamicClient, error) {
	memberConfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "members.config")

	var targetContext string
	if memberLabel == "private" {
		targetContext = "member1"
	} else if memberLabel == "public" {
		targetContext = "member2"
	} else {
		return nil, fmt.Errorf("unsupported member label: %s. Use 'private' or 'public'.", memberLabel)
	}

	// Carregar o arquivo kubeconfig e construir a configuração para o contexto específico
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: memberConfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: targetContext,
		}).ClientConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to build config from %s for context %s: %w", memberConfigPath, targetContext, err)
	}

	config.QPS = 100
	config.Burst = 200

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client for %s (via %s): %w", targetContext, memberConfigPath, err)
	}
	return dynClient, nil
}