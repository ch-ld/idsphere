package routers

import (
	"github.com/gin-gonic/gin"
	"ops-api/controller"
)

// 初始化菜单相关路由
func initHostRouter(router *gin.Engine) {
	// 获取主机列表
	router.GET("/api/v1/hosts", controller.Host.HostList)
	// 创建主机
	router.POST("/api/v1/host", controller.Host.HostCreate)
	// 更新主机
	router.PUT("/api/v1/host/:id", controller.Host.HostUpdate)
	// 删除主机
	router.DELETE("/api/v1/host/:id", controller.Host.HostDelete)
	// 批量删除主机
	router.POST("/api/v1/hosts/deletes", controller.Host.HostsBatchDelete)
	// 导入主机
	router.POST("/api/v1/hosts/import", controller.Host.HostsImport)
	// 获取主机状态
	router.GET("/api/v1/host/check/:id", controller.Host.HostsCheck)
	// 同步云主机
	router.POST("/api/v1/hosts/sync", controller.Host.HostsSyncCloud)
}
