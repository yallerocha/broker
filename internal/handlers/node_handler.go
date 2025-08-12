package eventhandler

import (
    "fmt"
    "strconv"
    "strings"

    "github.com/cloud-ai-ufcg/broker/internal/events"
    "github.com/cloud-ai-ufcg/broker/pkg/utils"
    "github.com/go-gota/gota/dataframe"
    "k8s.io/client-go/dynamic"
)

// Node_action identifies the specific action related to a given Kubernetes Node and performs it.
// It receives a dynamic client (from Karmada, though it will be ignored for direct member cluster connection),
// a DataFrame containing the event data, and the index of the current node event in the DataFrame.
func Node_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
    name := df.Col("id").Elem(idx).String()
    cpuRequested := df.Col("cpu").Elem(idx).String() // CPU for Nodes is typically in integer cores (e.g., "2", "4"), passed as string.

    // --- Memory Logic for Node ---
    // Attempts to parse memory as a float for numerical values (e.g., "4" for 4GiB).
    // If parsing fails (indicating a string with a unit like "4Gi" or "4096Mi"), it uses the raw string.
    memInput := df.Col("memory").Elem(idx).String()
    var memRequested string
    if memFloat, err := strconv.ParseFloat(memInput, 64); err == nil {
        // If it's a number (assumed to be in GB), format it as a string with "Gi".
        // E.g., "4" GB becomes "4Gi" GiB.
        memRequested = fmt.Sprintf("%dGi", int64(memFloat))
    } else {
        // If it's already a K8s format string (e.g., "4Gi"), use it as is.
        memRequested = memInput
    }

    label := df.Col("label").Elem(idx).String() // The label is crucial for selecting the target member cluster.
    action := strings.ToLower(df.Col("action").Elem(idx).String())

    // Attempt to get a DynamicClient specific to the member cluster associated with the label.
    // This ensures nodes are created directly in member1 or member2, not via Karmada's control plane.
    nodeDynamicContext, err := utils.GetDynamicClientForMemberCluster(label)
    if err != nil {
        utils.Log_err(fmt.Sprintf("Failed to get dynamic client for node label '%s': %v", label, err), err)
        return
    }

    nodeWorkload := utils.Workload{
        Name:         name,
        Replicas:     1, // Nodes are single units; replicas is typically 1 for them.
        CpuRequested: cpuRequested,
        MemRequested: memRequested,
        Label:        label,
        Annotations:  map[string]string{}, // Kwok-specific annotations are added during manifest creation.
    }

    utils.Log_info(fmt.Sprintf("➡️ [%ss] [Node] %s: %s (CPU: %s, Mem: %s) on cluster for label '%s'", df.Col("timestamp").Elem(idx).String(), strings.ToUpper(action), nodeWorkload.Name, nodeWorkload.CpuRequested, nodeWorkload.MemRequested, label))

    switch action {
    case "create":
        err := events.Create_Workload(nodeDynamicContext, nodeWorkload, "core", "nodes")
        utils.Log_err("Failed to create Node", err)
    case "delete":
        // TODO: Implement "delete" action for nodes using nodeDynamicContext.
        utils.Log_err(fmt.Sprintf("Node delete action not implemented yet! Action: %s for node: %s", action, name), fmt.Errorf(""))
    case "update":
        // TODO: Implement "update" action for nodes using nodeDynamicContext.
        utils.Log_err(fmt.Sprintf("Node update action not implemented yet! Action: %s for node: %s", action, name), fmt.Errorf(""))
    default:
        utils.Log_err(fmt.Sprintf("Unknown action: %s", action), fmt.Errorf("dataframe line %d", idx+2))
    }
}