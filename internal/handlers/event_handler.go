package eventhandler

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"k8s.io/client-go/dynamic"
)

// Define execution modes for the event handler.
const (
	ModeInit       = "init"       // Initialization mode: prioritizes node creation
	ModeSimulation = "simulation" // Simulation mode: processes events sequentially by timestamp
	ModeCLI        = "cli"        // Command-Line Interface mode: processes events sequentially (similar to simulation)
)

// Handler is the entrypoint to process a dataframe of events.
// It receives the event data as a DataFrame and the execution mode.
// The mode determines the order of event processing (e.g., node prioritization in "init" mode).
func Handler(origin_data dataframe.DataFrame, mode string, config utils.Config) {
	var karmadaDynamicContext *dynamic.DynamicClient
	var morpheusClient *utils.MorpheusClient
	var err error
	switch config.Orchestrator {
	case "morpheus":
		morpheusClient, err = utils.NewMorpheusClient()
		if err != nil {
			utils.Log_fatal("Failed to create Morpheus client", err)
		}
		if morpheusClient == nil || morpheusClient.Client == nil {
			utils.Log_fatal("Morpheus client is nil after creation", fmt.Errorf("nil morpheus client"))
		}
	case "karmada":
		karmadaDynamicContext, err = utils.GetDynamicContext()
		utils.Log_fatal("Failed to get Karmada client context", err) // Fatal if Karmada client cannot be obtained
	}

	start_time := time.Now()
	df := origin_data.Arrange(dataframe.Sort("timestamp")) // Sort events by timestamp
	rows := df.Records()

	utils.Log_info(fmt.Sprintf("DEBUG: Total rows in dataframe: %d. Mode: %s", df.Nrow(), mode))
	if mode == ModeInit {
		if config.Orchestrator != "karmada" {
			utils.Log_info("INFO: Node creation is only supported with Karmada orchestrator. Skipping.")
		} else {
			utils.Log_info("INFO: Running in INIT mode. Prioritizing node creation.")
			// Phase 1: Process only "Node" events to ensure infrastructure is ready.
			for i := range rows[0:df.Nrow()] {
				kind := df.Col("kind").Elem(i).String()
				id := df.Col("id").Elem(i).String()
				// Process only nodes in the first phase.
				if strings.ToLower(kind) == "node" {
					utils.Log_info(fmt.Sprintf("DEBUG: Processing INIT Phase 1 (Node): Kind='%s', ID='%s'", kind, id))
					// Node_action will obtain its specific member cluster client internally.
					// karmadaDynamicContext is passed but not directly used by Node_action.
					Node_action(karmadaDynamicContext, df, i)
				}
			}
		}

		utils.Log_info("INFO: INIT Phase 1 (Nodes) completed. Starting Phase 2 (Workloads).")
		// Phase 2: Process other events (Deployments, Jobs) after nodes are handled.
		if config.Orchestrator == "morpheus" {
			for i := range rows[0:df.Nrow()] {
				kind := df.Col("kind").Elem(i).String()
				id := df.Col("id").Elem(i).String()
				utils.Log_info(fmt.Sprintf("DEBUG: Processing INIT Phase 2 (Deployment): Kind='%s', ID='%s'", kind, id))
				Morpheus_Action(morpheusClient, df, i)
			}

		}
		for i := range rows[0:df.Nrow()] {
			kind := df.Col("kind").Elem(i).String()
			id := df.Col("id").Elem(i).String()
			// Process only Deployments and Jobs (excluding Nodes).
			if strings.ToLower(kind) == "deployment" {
				utils.Log_info(fmt.Sprintf("DEBUG: Processing INIT Phase 2 (Deployment): Kind='%s', ID='%s'", kind, id))
				// sleep_time(time_stamp, start_time) // Re-add if you want sleep in init phase 2
				Deployment_action(karmadaDynamicContext, df, i)
			} else if strings.ToLower(kind) == "job" {
				utils.Log_info(fmt.Sprintf("DEBUG: Processing INIT Phase 2 (Job): Kind='%s', ID='%s'", kind, id))
				// sleep_time(time_stamp, start_time) // Re-add if you want sleep in init phase 2
				Job_action(karmadaDynamicContext, df, i)
			}
		}
		utils.Log_info("INFO: INIT Phase 2 (Workloads) completed.")

	} else {
		utils.Log_info(fmt.Sprintf("INFO: Running in %s mode. Processing events sequentially.", mode))
		for i := range rows[0:df.Nrow()] {
			kind := df.Col("kind").Elem(i).String()
			id := df.Col("id").Elem(i).String()
			utils.Log_info(fmt.Sprintf("DEBUG: Processing row %d: Kind='%s', ID='%s'", i, kind, id))

			time_stamp, err := df.Col("timestamp").Elem(i).Int()
			utils.Log_err("Failed to read the timestamp", err)

			// Apply sleep based on timestamp to simulate real-time event flow.
			sleep_time(time_stamp, start_time)

			switch config.Orchestrator {
			case "karmada":
				if strings.ToLower(kind) == "deployment" {
					utils.Log_info("DEBUG: Matched kind: deployment")
					Deployment_action(karmadaDynamicContext, df, i)
				} else if strings.ToLower(kind) == "job" {
					utils.Log_info("DEBUG: Matched kind: job")
					Job_action(karmadaDynamicContext, df, i)
				} else if strings.ToLower(kind) == "node" {
					utils.Log_info("DEBUG: Matched kind: node")
					// Node_action will obtain its specific member cluster client internally.
					// karmadaDynamicContext is passed but not directly used by Node_action.
					Node_action(karmadaDynamicContext, df, i)
				} else {
					utils.Log_err(fmt.Sprintf("Unknown kind '%s'", kind), fmt.Errorf("dataframe line %d", i+2))
				}
			case "morpheus":
				if strings.ToLower(kind) == "deployment" || strings.ToLower(kind) == "job" {
					Morpheus_Action(morpheusClient, df, i)
				} else {
					utils.Log_err(fmt.Sprintf("Unknown kind '%s' for Morpheus orchestrator", kind), fmt.Errorf("dataframe line %d", i+2))
				}
			default:
				utils.Log_err(fmt.Sprintf("Unsupported orchestrator '%s'", config.Orchestrator), fmt.Errorf(""))
			}
		}
	}
}

// sleep_time synchronizes the broker execution with the 'timestamp' defined in the dataframe.
// If the current elapsed time is less than the event's timestamp, it pauses execution
// for the duration of the difference.
// It receives the current event's 'time_stamp' and the 'start_time' of the handler execution.
func sleep_time(time_stamp int, start_time time.Time) {
	elapsed := time.Since(start_time)

	if int64(math.Ceil(elapsed.Seconds())) < int64(time_stamp) {
		time_wait := float64(time_stamp) - elapsed.Seconds()

		utils.Log_info(fmt.Sprintf("⏳ Waiting %d seconds", int64(time_wait)))
		time.Sleep(time.Duration(time_wait) * time.Second)
	}
}
