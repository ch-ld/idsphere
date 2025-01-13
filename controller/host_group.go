package controller

import (
	"github.com/gin-gonic/gin"
	"ops-api/service"
	"strconv"
)

var HostGroup hostGroup

type hostGroup struct{}

// 列表查询
type ListHostGroupsInput struct {
	Name  string `form:"name"`
	Page  int    `form:"page" binding:"required,min=1"`
	Limit int    `form:"limit" binding:"required,min=1,max=100"`
}

//// 获取单个主机组
//func (u *hostGroup) GetHostGroup(c *gin.Context) {
//	id, _ := strconv.Atoi(c.Param("id"))
//	data, err := service.HostGroup.GetByID(uint(id))
//	if err != nil {
//		Response(c, 90500, err.Error())
//		return
//	}
//
//	c.JSON(200, gin.H{
//		"code": 0,
//		"data": data,
//	})
//}

// GetHostGroupList 获取主机组列表
// @Summary 获取主机组列表
// @Description 主机组相关接口
// @Tags 主机组管理
// @Param Authorization header string true "Bearer 用户令牌"
// @Param page query int true "分页"
// @Param limit query int true "分页大小"
// @Param name query string false "主机组名称"
// @Success 200 {string} json "{"code": 0, "data": [], "total": 0}"
// @Router /api/v1/host-groups [get]
// 获取主机组列表
func (hg *hostGroup) ListHostGroups(c *gin.Context) {
	var input ListHostGroupsInput
	if err := c.ShouldBind(&input); err != nil {
		Response(c, 400, err.Error())
		return
	}

	data, total, err := service.HostGroup.List(input.Name, input.Page, input.Limit)
	if err != nil {
		Response(c, 90500, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"code":  0,
		"data":  data,
		"total": total,
	})
}

// CreateHostGroup 创建主机组
// @Summary 创建主机组
// @Description 创建一个新的主机组
// @Tags 主机组管理
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param name query string true "主机组名"
// @Param description query string true "主机组描述信息"
// @Success 200 {string} json "{"code": 0, "msg": "主机组创建成功"}"
// @Router /api/v1/host-group [post]
// 创建主机组
func (hg *hostGroup) CreateHostGroup(c *gin.Context) {
	var input service.HostGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Response(c, 400, err.Error())
		return
	}

	err := service.HostGroup.Create(&input)
	if err != nil {
		Response(c, 500, err.Error())
		return
	}
	Response(c, 0, "主机组创建成功")
}

// UpdateHostGroup 更新主机组
// @Summary 更新主机组
// @Description 更新一个新的主机组
// @Tags 主机组管理
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param name query string true "主机组名"
// @Param description query string true "主机组描述信息"
// @Success 200 {string} json "{"code": 0, "msg": "主机组更新成功"}"
// @Router /api/v1/host-group/:id [put]
// 更新主机组
func (hg *hostGroup) UpdateHostGroup(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var input service.HostGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Response(c, 400, err.Error())
		return
	}

	err := service.HostGroup.Update(uint(id), &input)
	if err != nil {
		Response(c, 500, err.Error())
		return
	}
	Response(c, 0, "主机组更新成功")
}

// DeleteHostGroup 删除主机组
// @Summary 删除主机组
// @Description 删除主机组
// @Tags 主机组管理
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "主机组id"
// @Success 200 {string} json "{"code": 0, "msg": "主机组删除成功"}"
// @Router /api/v1/host-group/{id} [delete]
// 删除主机组
func (hg *hostGroup) DeleteHostGroup(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := service.HostGroup.Delete(uint(id))
	if err != nil {
		Response(c, 500, err.Error())
		return
	}
	Response(c, 0, "主机组删除成功")
}

// BatchDeleteHostGroup 批量删除主机组
// @Summary 批量删除主机组
// @Description 批量删除主机组
// @Tags 主机组管理
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param ids body []int true "主机组id列表"
// @Success 200 {string} json "{"code": 0, "msg": "主机组批量删除成功"}"
// @Router /api/v1/host-groups/batch-delete [delete]
// 批量删除主机组
func (hg *hostGroup) BatchDeleteHostGroup(c *gin.Context) {
	var ids []uint
	if err := c.ShouldBindJSON(&ids); err != nil {
		Response(c, 400, err.Error())
		return
	}

	err := service.HostGroup.BatchDelete(ids)
	if err != nil {
		Response(c, 500, err.Error())
		return
	}
	Response(c, 0, "主机组批量删除成功")
}
