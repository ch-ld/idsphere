package routers

import (
	"github.com/gin-gonic/gin"
	"ops-api/controller"
)

// 初始化菜单相关路由
func initMenuRouters(router *gin.Engine) {
	// 获取菜单列表
	router.GET("/api/v1/menus", controller.Menu.GetMenuList)
	// 获取接口列表
	router.GET("/api/v1/paths", controller.Path.GetPathList)
	// 创建菜单
	router.POST("/api/v1/menus", controller.Menu.CreateMenu)
	// 更新菜单
	router.PUT("/api/v1/menus/:id", controller.Menu.UpdateMenu)
	// 删除菜单
	router.DELETE("/api/v1/menus/:id", controller.Menu.DeleteMenu)
	// 更新菜单排序
	router.PUT("/api/v1/menus/:id/sort", controller.Menu.UpdateMenuSort)

	// 子菜单路由
	// 获取创建子菜单
	router.POST("/api/v1/submenus", controller.Menu.CreateSubMenu)
	// 更新子菜单
	router.PUT("/api/v1/submenus/:id", controller.Menu.UpdateSubMenu)
	// 删除子菜单
	router.DELETE("/api/v1/submenus/:id", controller.Menu.DeleteSubMenu)
	// 更新子菜单排序
	router.PUT("/api/v1/submenus/:id/sort", controller.Menu.UpdateSubMenuSort)
}
