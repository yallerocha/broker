package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
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
func NewMorpheusClient(morpheus_config MorpheusConfig) (*MorpheusClient, error) {
	client := morpheus.NewClient(morpheus_config.URL)
	client.SetAccessToken(morpheus_config.AccessToken, morpheus_config.RefreshToken, morpheus_config.ExpiresIn, morpheus_config.Scope)

	// Attach debug HTTP transport by replacing the default transport with a
	// wrapper that logs requests/responses. This is a temporary debugging aid
	// because the gomorpheus client doesn't expose a public HTTPClient field.
	// We only replace it for debugging purposes; in production you may want a
	// cleaner approach.
	dbg := &debugRoundTripper{rt: http.DefaultTransport}
	http.DefaultTransport = dbg

	return &MorpheusClient{Client: client}, nil
}

// debugRoundTripper logs basic request and response info and truncates large bodies.
type debugRoundTripper struct {
	rt http.RoundTripper
}

func (d *debugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request method and path
	fmt.Printf("[MORPHEUS-DEBUG] Request: %s %s\n", req.Method, req.URL)

	if req.Body != nil {
		rb, _ := ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(rb))
		if len(rb) > 8192 {
			fmt.Printf("[MORPHEUS-DEBUG] Request Body (truncated to 8KB): %s...\n", string(rb[:8192]))
		} else {
			fmt.Printf("[MORPHEUS-DEBUG] Request Body: %s\n", string(rb))
		}
	}

	resp, err := d.rt.RoundTrip(req)
	if err != nil {
		fmt.Printf("[MORPHEUS-DEBUG] RoundTrip error: %v\n", err)
		return resp, err
	}

	if resp != nil && resp.Body != nil {
		rb, _ := ioutil.ReadAll(resp.Body)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(rb))
		if len(rb) > 8192 {
			fmt.Printf("[MORPHEUS-DEBUG] Response Status: %s Body (truncated): %s...\n", resp.Status, string(rb[:8192]))
		} else {
			fmt.Printf("[MORPHEUS-DEBUG] Response Status: %s Body: %s\n", resp.Status, string(rb))
		}
	} else {
		fmt.Printf("[MORPHEUS-DEBUG] Response Status: %v (no body)\n", resp.Status)
	}

	return resp, nil
}
