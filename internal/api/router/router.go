package router

import "github.com/gin-gonic/gin"

func Init_router() {
	router := gin.Default()
	execution_routes(router)

	router.Run()
}
