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
	KubeConfig string `yaml:"kubeconfig"`
	Namespace  string `yaml:"namespace"`
}

type MorpheusClient struct {
	client *morpheus.Client
}

type MorpheusConfig struct {
	URL          string `yaml:"url"`
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	ExpiresIn    int64  `yaml:"expires_in"`
	Scope        string `yaml:"scope"`
}
