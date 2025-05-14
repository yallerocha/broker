package utils

type Workload struct {
	Name         string
	Replicas     int32
	CpuRequested string
	MemRequested string
	Label        string
	Annotations  map[string]string
}
