package handlers

import (
	"github.com/gin-gonic/gin"
)

func Start_broker(ctx *gin.Context) {
	ctx.IndentedJSON(200, nil)
}
