package router

import "github.com/gin-gonic/gin"

// Initiate the http server using the port 8081 and its routes using gin.
func Init_router() {
	router := gin.Default()
	execution_routes(router)

	router.Run(":8081")
}
