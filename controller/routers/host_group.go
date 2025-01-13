package routers

import (
	"github.com/gin-gonic/gin"
	"ops-api/controller"
)

// 初始化菜单相关路由
func initHostGroupRouters(router *gin.Engine) {
	// 获取主机组列表
	router.GET("/api/v1/host-groups", controller.HostGroup.ListHostGroups)
	// 创建主机组
	router.POST("/api/v1/host-group", controller.HostGroup.CreateHostGroup)
	// 更新主机组
	router.PUT("/api/v1/host-group/:id", controller.HostGroup.UpdateHostGroup)
	// 删除主机组
	router.DELETE("/api/v1/host-group/:id", controller.HostGroup.DeleteHostGroup)
	// 批量删除主机组
	router.POST("/api/v1/host-groups/batch-delete", controller.HostGroup.BatchDeleteHostGroup)
}
