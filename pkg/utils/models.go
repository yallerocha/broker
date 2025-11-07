package utils

type Workload struct {
	Name         string
	Replicas     int32
	CpuRequested string
	MemRequested string
	Label        string
	WorkloadType string // Type of workload: cpu-intensive, memory-intensive, io-intensive, network-intensive, mixed, bursty, idle
	Annotations  map[string]string
}

type Config struct {
	KubeConfig string `yaml:"kubeconfig"`
	Namespace  string `yaml:"namespace"`
}
