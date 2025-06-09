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

var (
	g         = galidator.New()
	validator = g.Validator(dto.Start_request{})
)

// Defines a handler function to start broker route.
// It can return two http status code:
//   - 200 -> the broker was successful
//   - 400 -> the request has an error
func Start_broker(ctx *gin.Context) {
	var start_dto dto.Start_request

	if err := ctx.ShouldBindJSON(&start_dto); err != nil {
		ctx.IndentedJSON(400, gin.H{"message": validator.DecryptErrors(err)})
		return
	}

	broker.Run_from_api(to_dataframe(start_dto), to_config(start_dto))

	ctx.IndentedJSON(200, gin.H{"message": "The broker was successful"})
}

// Converts the dto struct to a dataframe.
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

// Converts the Start dto to Config struct.
func to_config(body dto.Start_request) utils.Config {
	config := utils.Config{
		KubeConfig: body.Config.Kubeconfig,
		Namespace:  body.Config.Namespace,
	}

	return config
}
