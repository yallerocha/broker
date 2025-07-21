package router

import (
	"github.com/cloud-ai-ufcg/broker/internal/api/handlers"
	"github.com/gin-gonic/gin"
)

// Create and handle multiple routes related to the execution routine.
func execution_routes(router *gin.Engine) {
	group := router.Group("/broker")
	{
		group.POST("/", handlers.Start_broker)
		group.POST("/init", handlers.Init_broker)
		group.POST("/simulation", handlers.Simulation_broker)
	}

}
