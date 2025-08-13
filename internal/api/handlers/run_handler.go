package handlers

import (
	"strings"

	"github.com/cloud-ai-ufcg/broker/broker"
	"github.com/cloud-ai-ufcg/broker/internal/api/dto"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-gota/gota/dataframe"
	"github.com/golodash/galidator/v2"
)

// Global validator instance using galidator for Start_request DTO.
var (
	g         = galidator.New()
	validator = g.Validator(dto.Start_request{})
)

// Start_broker defines a handler function for the base broker route ("/broker/").
// It handles API requests to run the broker in a default "simulation" mode.
// It returns HTTP status 200 on success or 400 on bad request.
func Start_broker(ctx *gin.Context) {
	var start_dto dto.Start_request

	if err := ctx.ShouldBindJSON(&start_dto); err != nil {
		ctx.IndentedJSON(400, gin.H{"message": validator.DecryptErrors(err)})
		return
	}

	// Run the broker's main logic in "simulation" mode.
	broker.Run_from_api(to_dataframe(start_dto), to_config(start_dto), "simulation")

	ctx.IndentedJSON(200, gin.H{"message": "The broker was successful"})
}

// Init_broker defines a handler function for the "/broker/init" route.
// This route is specifically designed for initial setup, such as creating Kwok nodes.
// It runs the broker in "init" mode, which prioritizes node creation before other workloads.
// It returns HTTP status 200 on success or 400 on bad request.
func Init_broker(ctx *gin.Context) {
	var init_dto dto.Start_request

	if err := ctx.ShouldBindJSON(&init_dto); err != nil {
		ctx.IndentedJSON(400, gin.H{"message": validator.DecryptErrors(err)})
		return
	}

	// Run the broker's main logic in "init" mode.
	broker.Run_from_api(to_dataframe(init_dto), to_config(init_dto), "init")

	ctx.IndentedJSON(200, gin.H{"message": "The broker initialization was successful"})
}

// Simulation_broker defines a handler function for the "/broker/simulation" route.
// This route is intended for running the main simulation logic.
// It runs the broker in "simulation" mode, processing events based on timestamps.
// It returns HTTP status 200 on success or 400 on bad request.
func Simulation_broker(ctx *gin.Context) {
	var simulation_dto dto.Start_request

	// Bind the JSON request body to the Start_request DTO and validate.
	if err := ctx.ShouldBindJSON(&simulation_dto); err != nil {
		ctx.IndentedJSON(400, gin.H{"message": validator.DecryptErrors(err)})
		return
	}

	broker.Run_from_api(to_dataframe(simulation_dto), to_config(simulation_dto), "simulation")

	ctx.IndentedJSON(200, gin.H{"message": "The broker simulation was successful"})
}

// to_dataframe converts the 'Data' field from the Start_request DTO into a Gota DataFrame.
// It renames column headers to lowerCamelCase for consistency with Go DataFrame conventions.
func to_dataframe(body dto.Start_request) *dataframe.DataFrame {
	df := dataframe.LoadStructs(body.Data)
	names := df.Names()

	for _, name := range names {
		old_name := name
		new_name := strings.ToLower(name[:1]) + name[1:]

		df = df.Rename(new_name, old_name)
	}

	return &df
}

// to_config converts relevant fields from the Start_request DTO into a utils.Config struct.
// This extracts Kubernetes configuration details (Kubeconfig path and Namespace).
func to_config(body dto.Start_request) utils.Config {
	config := utils.Config{
		KubeConfig: body.Config.Kubeconfig,
		Namespace:  body.Config.Namespace,
	}

	return config
}