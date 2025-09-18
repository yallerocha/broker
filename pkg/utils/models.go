package utils

import "github.com/gomorpheus/morpheus-go-sdk"

type Workload struct {
	Name         string
	Replicas     int32
	CpuRequested string
	MemRequested string
	Label        string
	Annotations  map[string]string
}

type Config struct {
	Orchestrator         string `yaml:"orchestrator"`
	KarmadaKubeConfig    string `yaml:"karmada_kubeconfig"`
	KarmadaNamespace     string `yaml:"karmada_namespace"`
	MorpheusURL          string `yaml:"morpheus_url"`
	MorpheusAccessToken  string `yaml:"morpheus_access_token"`
	MorpheusRefreshToken string `yaml:"morpheus_refresh_token"`
	MorpheusExpiresIn    int64  `yaml:"morpheus_expires_in"`
	MorpheusScope        string `yaml:"morpheus_scope"`
}

type MorpheusClient struct {
	Client *morpheus.Client
}

type MorpheusConfig struct {
	URL          string `yaml:"url"`
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	ExpiresIn    int64  `yaml:"expires_in"`
	Scope        string `yaml:"scope"`
}
