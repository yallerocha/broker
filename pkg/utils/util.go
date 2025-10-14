package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gomorpheus/morpheus-go-sdk"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	orchestrator           string
	karmada_kubepath       string
	karmada_namespace      string
	morpheus_access_token  string
	morpheus_refresh_token string
	morpheus_expires_in    int64
	morpheus_scope         string
	morpheus_url           string
)

// Set the kubernetes context path. Este continua sendo usado para configurar o caminho do kubeconfig principal (do Karmada).
// Ele é chamado na inicialização do broker (geralmente em main.go).
func Set_context(config Config) {
	orchestrator = config.Orchestrator
	if orchestrator == "" {
		orchestrator = "karmada" // Default to Karmada if not specified
	}

	Log_info(fmt.Sprintf("Orchestrator set to: %s", orchestrator))

	if orchestrator == "karmada" {
		karmada_kubepath = filepath.Join(
			os.Getenv("HOME"), ".kube", config.KarmadaKubeConfig, // Assume que karmada.config está em ~/.kube/
		)
		Log_info(fmt.Sprintf("Kubeconfig path for default context (Karmada) set to: %s", karmada_kubepath))
	}

	karmada_namespace = config.KarmadaNamespace
	morpheus_url = config.MorpheusURL
	morpheus_access_token = config.MorpheusAccessToken
	morpheus_refresh_token = config.MorpheusRefreshToken
	morpheus_expires_in = config.MorpheusExpiresIn
	morpheus_scope = config.MorpheusScope
}

// Get the target namespace
func Get_context_namespace() string {
	return karmada_namespace
}

// GetDynamicContext retorna um DynamicClient para o contexto principal do broker (o Karmada).
// Esta é a função original, que workloads usarão.
func GetDynamicContext() (*dynamic.DynamicClient, error) {
	if karmada_kubepath == "" {
		// Fallback ou erro se o kubeconfig do Karmada não for configurado.
		return nil, fmt.Errorf("karmada kubeconfig path not set. Call Set_context_path first.")
	}
	config, err := clientcmd.BuildConfigFromFlags("", karmada_kubepath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from flags for Karmada (%s): %w", karmada_kubepath, err)
	}

	config.QPS = 100
	config.Burst = 200

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client for Karmada (%s): %w", karmada_kubepath, err)
	}
	return dynClient, nil
}

// GetDynamicClientForMemberCluster retorna um DynamicClient para um cluster membro específico.
// Ele usa o 'members.config' e seleciona o contexto apropriado dentro dele.
// Esta é a nova função, usada especificamente para Nodes.
func GetDynamicClientForMemberCluster(memberLabel string) (*dynamic.DynamicClient, error) {
	memberConfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "members.config")

	var targetContext string
	switch memberLabel {
	case "private":
		targetContext = "member1"
	case "public":
		targetContext = "member2"
	default:
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

func SetMorpheusConfig(url, accessToken, refreshToken string, expiresIn int64, scope string) *MorpheusConfig {
	return &MorpheusConfig{
		URL:          url,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Scope:        scope,
	}
}

// NewMorpheusClient creates a new Morpheus orchestrator instance
// morpheusURL, accessToken, refreshToken, expiresIn, and scope are required for authentication
func NewMorpheusClient() (*MorpheusClient, error) {
	morpheusConfig := MorpheusConfig{
		URL:          morpheus_url,
		AccessToken:  morpheus_access_token,
		RefreshToken: morpheus_refresh_token,
		ExpiresIn:    morpheus_expires_in,
		Scope:        morpheus_scope,
	}
	client := morpheus.NewClient(morpheus_url)
	client.SetAccessToken(morpheus_access_token, morpheus_refresh_token, morpheus_expires_in, morpheus_scope)

	fmt.Println(client)
	return &MorpheusClient{Client: client, Config: morpheusConfig}, nil
}
