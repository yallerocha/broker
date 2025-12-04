package utils

import "github.com/gomorpheus/morpheus-go-sdk"

type Workload struct {
	Name         string
	Replicas     int32
	CpuRequested string
	MemRequested string
	Label        string
	// WorkloadType defines the behavior of the workload container.
	// Available types:
	//   - cpu-intensive: Constant high CPU usage (default)
	//   - memory-intensive: Allocates and holds memory (85% of requested)
	//   - io-intensive: Heavy disk I/O operations
	//   - network-intensive: Socket/network stress
	//   - mixed: Combination of CPU, memory, and I/O
	//   - bursty: Alternates between high CPU (10s) and idle (20s)
	//   - idle: Minimal resource usage, just stays alive
	//   - realistic: Variable load pattern simulating real web app traffic
	//   - microservice: Network-heavy with occasional CPU spikes
	//   - database: I/O and memory intensive with query simulation
	//   - batch: Heavy processing followed by idle periods
	WorkloadType string
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
	Config MorpheusConfig
}

type MorpheusConfig struct {
	URL          string `yaml:"url"`
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	ExpiresIn    int64  `yaml:"expires_in"`
	Scope        string `yaml:"scope"`
}
