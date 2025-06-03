package router

import (
	"github.com/cloud-ai-ufcg/broker/internal/api/handlers"
	"github.com/gin-gonic/gin"
)

func execution_routes(router *gin.Engine) {
	group := router.Group("/broker")
	{
		group.POST("/", handlers.Start_broker)
	}
}
