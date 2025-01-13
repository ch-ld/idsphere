package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/wonderivan/logger"
	"net/http"
	"ops-api/model"
	"ops-api/service"
	"strconv"
)

var Menu menu

type menu struct{}

// GetMenuListAll 获取所菜单
// @Summary 获取所菜单
// @Description 组相关接口
// @Tags 组管理
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/menu/list [get]
func (u *menu) GetMenuListAll(c *gin.Context) {

	data, err := service.Menu.GetMenuListAll()

	if err != nil {
		logger.Error("ERROR：" + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": data,
	})
}

// GetMenuList 获取菜单列表
// @Summary 获取菜单列表
// @Description 菜单关接口
// @Tags 菜单管理
// @Param Authorization header string true "Bearer 用户令牌"
// @Param page query int true "分页"
// @Param limit query int true "分页大小"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/menus [get]
func (u *menu) GetMenuList(c *gin.Context) {
	params := new(struct {
		Title string `form:"title"`
		Page  int    `form:"page" binding:"required"`
		Limit int    `form:"limit" binding:"required"`
	})
	if err := c.Bind(params); err != nil {
		logger.Error("ERROR：" + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	data, err := service.Menu.GetMenuList(params.Title, params.Page, params.Limit)
	if err != nil {
		logger.Error("ERROR：" + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": data,
	})
}

// CreateMenu 创建菜单
// @Summary 创建菜单
// @Description 创建新的菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param menu body model.Menu true "菜单信息"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/menus [post]
func (u *menu) CreateMenu(c *gin.Context) {
	menu := new(model.Menu)
	if err := c.ShouldBindJSON(menu); err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	if err := service.Menu.CreateMenu(menu); err != nil {
		logger.Error("创建菜单失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "创建成功",
	})
}

// CreateSubMenu 创建子菜单
// @Summary 创建子菜单
// @Description 创建新的子菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param subMenu body model.SubMenu true "子菜单信息"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/submenus [post]
func (u *menu) CreateSubMenu(c *gin.Context) {
	subMenu := new(model.SubMenu)
	if err := c.ShouldBindJSON(subMenu); err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	if err := service.Menu.CreateSubMenu(subMenu); err != nil {
		logger.Error("创建子菜单失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "创建成功",
	})
}

// UpdateMenu 更新菜单
// @Summary 更新菜单
// @Description 更新现有菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path uint true "菜单ID"
// @Param menu body map[string]interface{} true "更新信息"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/menus/{id} [put]
func (u *menu) UpdateMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}
	updates := make(map[string]interface{})
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	if err := service.Menu.UpdateMenu(uint(id), updates); err != nil {
		logger.Error("更新菜单失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
	})
}

// DeleteMenu 删除菜单
// @Summary 删除菜单
// @Description 删除现有菜单
// @Tags 菜单管理
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path uint true "菜单ID"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/menus/{id} [delete]
func (u *menu) DeleteMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	if err := service.Menu.DeleteMenu(uint(id)); err != nil {
		logger.Error("删除菜单失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "删除成功",
	})
}

// UpdateMenuSort 更新菜单排序
// @Summary 更新菜单排序
// @Description 更新菜单的排序
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path uint true "菜单ID"
// @Param sort body int true "排序值"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/menus/{id}/sort [put]
func (u *menu) UpdateMenuSort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}
	var sort struct {
		Sort int `json:"sort" binding:"required"`
	}
	if err := c.ShouldBindJSON(&sort); err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	if err := service.Menu.UpdateMenuSort(uint(id), sort.Sort); err != nil {
		logger.Error("更新排序失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
	})
}

// UpdateSubMenu 更新子菜单
// @Summary 更新子菜单
// @Description 更新现有子菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path uint true "子菜单ID"
// @Param subMenu body map[string]interface{} true "更新信息"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/submenus/{id} [put]
func (u *menu) UpdateSubMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}
	updates := make(map[string]interface{})
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	if err := service.Menu.UpdateSubMenu(uint(id), updates); err != nil {
		logger.Error("更新子菜单失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
	})
}

// DeleteSubMenu 删除子菜单
// @Summary 删除子菜单
// @Description 删除现有子菜单
// @Tags 菜单管理
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path uint true "子菜单ID"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/submenus/{id} [delete]
func (u *menu) DeleteSubMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}
	if err := service.Menu.DeleteSubMenu(uint(id)); err != nil {
		logger.Error("删除子菜单失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "删除成功",
	})
}

// UpdateSubMenuSort 更新子菜单排序
// @Summary 更新子菜单排序
// @Description 更新子菜单的排序
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path uint true "子菜单ID"
// @Param sort body int true "排序值"
// @Success 200 {string} json "{"code": 0, "data": []}"
// @Router /api/v1/submenus/{id}/sort [put]
func (u *menu) UpdateSubMenuSort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}
	var sort struct {
		Sort int `json:"sort" binding:"required"`
	}
	if err := c.ShouldBindJSON(&sort); err != nil {
		logger.Error("参数解析错误: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90400,
			"msg":  err.Error(),
		})
		return
	}

	if err := service.Menu.UpdateSubMenuSort(uint(id), sort.Sort); err != nil {
		logger.Error("更新排序失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 90500,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
	})
}
