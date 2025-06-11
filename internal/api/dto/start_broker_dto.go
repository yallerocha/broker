package dto

// Data transfer object for requests in the 'post' route
type Start_request struct {
	Data   []row         `json:"data" binding:"required,min=1,dive,required" required:"$field is required"`
	Config broker_config `json:"config" binding:"required" required:"$field is required"`
}

// Defines a row representing the workload information
type row struct {
	Timestamp    int    `json:"timestamp" binding:"required"`
	Id           string `json:"id"`
	Kind         string `json:"kind" binding:"required"`
	Action       string `json:"action" binding:"required"`
	Replicas     string `json:"replicas"`
	Cpu          string `json:"cpu" binding:"required"`
	Memory       string `json:"memory" binding:"required"`
	Label        string `json:"label"`
	Job_duration string `json:"job_duration"`
}

// Defines a broker config containing the kubeconfig and the namespace information
type broker_config struct {
	Kubeconfig string `json:"kubeconfig" binding:"required"`
	Namespace  string `json:"namespace" binding:"required"`
}
