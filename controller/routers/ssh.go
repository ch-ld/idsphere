package routers

import (
	"github.com/gin-gonic/gin"
	"ops-api/controller"
)

func initSshRouters(router *gin.Engine) {
	// webssh终端
	router.GET("/api/v1/ssh/webssh", controller.SSH.WebSsh)
	// ssh执行命令
	router.POST("/api/v1/ssh/command", controller.SSH.SshCommand)
}
