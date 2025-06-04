package handlers

import (
	"github.com/cloud-ai-ufcg/broker/internal/api/dto"
	"github.com/gin-gonic/gin"
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

	ctx.IndentedJSON(200, gin.H{"message": "The broker was successful"})
}
