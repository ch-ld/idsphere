package routers

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"ops-api/config"
	"ops-api/docs"
)

var Router router

type router struct{}

func (r *router) InitRouter(router *gin.Engine) {

	// Swagger接口文档
	if config.Conf.Swagger {
		docs.SwaggerInfo.BasePath = ""
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	// 初始化不同类型路由
	initUserRouters(router)
	initGroupRouters(router)
	initSiteRouters(router)
	initAuditRouters(router)
	initSmsRouters(router)
	initMenuRouters(router)
	initAuthRouters(router)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})
}
