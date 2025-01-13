package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"ops-api/model"
	"ops-api/service"
	"ops-api/utils/msg"
	"ops-api/utils/result"
	"strconv"
	"strings"
)

// SyncCloudRequest 同步云主机请求结构
// @Description 同步云主机的请求参数
type SyncCloudRequest struct {
	// 云服务提供商（aliyun/aws）
	Provider string `json:"provider" binding:"required,oneof=aliyun aws" example:"aws"`
	// 访问密钥ID
	AccessKey string `json:"accessKey" binding:"required" example:"AKIAIOSFODNN7EXAMPLE"`
	// 访问密钥密文
	AccessSecret string `json:"accessSecret" binding:"required" example:"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`
	// 需要同步的区域列表
	Regions []string `json:"regions" binding:"required,min=1" example:"['us-west-1','us-east-1']"`
	// 主机组ID
	HostGroupId int `json:"hostGroupId" binding:"required" example:"1"`
}

// Host 主机信息
// @Description 主机基础信息
type HostResponse struct {
	// 主机ID
	ID uint `json:"id" example:"1"`
	// 创建时间
	CreatedAt string `json:"createdAt" example:"2024-01-07T15:04:05Z"`
	// 更新时间
	UpdatedAt string `json:"updatedAt" example:"2024-01-07T15:04:05Z"`
	// 主机名称
	Hostname string `json:"hostname" example:"web-server-01"`
	// 主机IP地址
	IP string `json:"ip" example:"192.168.1.100"`
	// 操作系统类型
	OsType string `json:"osType" example:"linux"`
	// 操作系统版本
	OsVersion string `json:"osVersion" example:"Ubuntu 20.04"`
	// CPU核心数
	CPU int `json:"cpu" example:"4"`
	// 内存大小(GB)
	Memory int `json:"memory" example:"8"`
	// 磁盘大小(GB)
	Disk int `json:"disk" example:"100"`
	// 主机状态
	Status string `json:"status" example:"running"`
	// 主机组ID
	HostGroupId uint `json:"hostGroupId" example:"1"`
	// 云服务商
	Provider string `json:"provider" example:"aws"`
	// 区域
	Region string `json:"region" example:"us-west-1"`
	// 实例类型
	InstanceType string `json:"instanceType" example:"t3.medium"`
	// 备注
	Comment string `json:"comment" example:"测试服务器"`
}

// HostListResponse 主机列表响应
// @Description 主机列表分页响应
type HostListResponse struct {
	// 主机列表
	List []HostResponse `json:"list"`
	// 总数
	Total int64 `json:"total" example:"100"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []int `json:"ids"`
}

var Host host

type host struct{}

// HostList 查询主机列表
// @Summary 查询主机列表
// @Description 获取主机列表，支持分页和筛选
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param keyword query string false "搜索关键词"
// @Param status query string false "主机状态"
// @Param hostGroupId query int false "主机组ID"
// @Success 200 {object} result.Result{data=HostListResponse} "成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts [get]
func (h *host) HostList(c *gin.Context) {
	var query service.ListHostInput
	if err := c.ShouldBindQuery(&query); err != nil {
		Response(c, http.StatusBadRequest, err.Error())
		return
	}
	hosts, total, err := service.Host.HostsList(&query)
	if err != nil {
		Response(c, http.StatusInternalServerError, err.Error())
		return
	}
	// 定义响应结构体
	type Response struct {
		List  []model.Host `json:"list"`
		Total int64        `json:"total"`
	}
	response := Response{
		List:  hosts,
		Total: total,
	}
	code := 200
	c.JSON(http.StatusOK, (&result.Result{}).Ok(code, response, msg.GetErrMsg(code)))
}

// Create 创建主机
// @Summary 创建主机
// @Description 创建新的主机记录
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param host body service.HostReq true "主机信息"
// @Success 200 {object} result.Result "成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts [post]
func (h *host) HostCreate(c *gin.Context) {
	var input service.HostReq
	if err := c.ShouldBindJSON(&input); err != nil {
		Response(c, http.StatusBadRequest, err.Error())
		return
	}
	err := service.Host.HostCreate(&input)
	if err != nil {
		Response(c, http.StatusInternalServerError, err.Error())
		return
	}

	code := 200
	c.JSON(http.StatusOK, (&result.Result{}).Ok(code, nil, msg.GetErrMsg(code)))
}

// Update 更新主机
// @Summary 更新主机
// @Description 更新现有主机信息
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "主机ID"
// @Param host body service.HostReq true "主机信息"
// @Success 200 {object} result.Result "成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts/{id} [put]
func (h *host) HostUpdate(c *gin.Context) {
	id := c.Param("id")
	var input service.HostReq
	if err := c.ShouldBindJSON(&input); err != nil {
		Response(c, http.StatusBadRequest, err.Error())
		return
	}
	hostId, _ := strconv.ParseUint(id, 10, 64)
	err := service.Host.HostUpdate(uint(hostId), &input)
	if err != nil {
		Response(c, http.StatusInternalServerError, err.Error())
		return
	}

	code := 200
	c.JSON(http.StatusOK, (&result.Result{}).Ok(code, nil, msg.GetErrMsg(code)))
}

// Delete 删除主机
// @Summary 删除主机
// @Description 删除指定ID的主机
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "主机ID"
// @Success 200 {object} result.Result "成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts/{id} [delete]
func (h *host) HostDelete(c *gin.Context) {
	id := c.Param("id")
	// 将字符串转换为 uint
	ID, _ := strconv.ParseUint(id, 10, 64)

	err := service.Host.HostDelete(uint(ID))
	if err != nil {
		Response(c, http.StatusInternalServerError, err.Error())
		return
	}
	code := 200
	c.JSON(http.StatusOK, (&result.Result{}).Ok(code, nil, msg.GetErrMsg(code)))
}

// BatchDelete 批量删除主机
// @Summary 批量删除主机
// @Description 批量删除多个主机
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param ids body BatchDeleteRequest true "主机ID列表"
// @Success 200 {object} result.Result "成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts/batch [delete]
func (h *host) HostsBatchDelete(c *gin.Context) {
	var input struct {
		IDs []int `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		Response(c, http.StatusBadRequest, err.Error())
		return
	}

	// 创建一个等长的 uint 切片
	uintIDs := make([]uint, len(input.IDs))

	// 转换每个元素
	for i, idInt := range input.IDs {
		uintIDs[i] = uint(idInt)
	}

	err := service.Host.HostBatchDelete(uintIDs)
	if err != nil {
		Response(c, http.StatusInternalServerError, err.Error())
		return
	}

	code := 200
	c.JSON(http.StatusOK, (&result.Result{}).Ok(code, nil, msg.GetErrMsg(code)))
}

// HostsImport 导入主机
// @Summary 导入主机
// @Description 通过文件导入主机信息（支持CSV、XLSX、XLS格式）
// @Tags 主机管理
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param file formData file true "导入文件（CSV/XLSX/XLS）"
// @Success 200 {object} result.Result "导入成功"
// @Failure 400 {object} result.Result "文件格式错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts/import [post]
func (h *host) HostsImport(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, (&result.Result{}).Error(msg.ERROR, nil, "请选择要上传的文件"))
		return
	}

	// 验证文件类型
	filename := strings.ToLower(file.Filename)
	if !strings.HasSuffix(filename, ".csv") &&
		!strings.HasSuffix(filename, ".xlsx") &&
		!strings.HasSuffix(filename, ".xls") {
		c.JSON(http.StatusBadRequest, (&result.Result{}).Error(msg.ERROR, nil, "只支持 CSV、XLSX 和 XLS 格式文件"))
		return
	}

	// 验证文件大小
	if file.Size > 10<<20 {
		c.JSON(http.StatusBadRequest, (&result.Result{}).Error(msg.ERROR, nil, "文件大小不能超过10MB"))
		return
	}

	fileReader, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, (&result.Result{}).Error(msg.ERROR, nil, "无法打开文件"))
		return
	}
	defer fileReader.Close()

	// 添加 panic 恢复
	defer func() {
		if r := recover(); r != nil {
			c.JSON(http.StatusInternalServerError, (&result.Result{}).Error(msg.ERROR, nil, fmt.Sprintf("导入过程发生错误: %v", r)))
		}
	}()

	err = service.Host.HostsImport(fileReader, file.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, (&result.Result{}).Error(msg.ERROR, nil, err.Error()))
		return
	}

	c.JSON(http.StatusOK, (&result.Result{}).Ok(msg.SUCCSE, nil, "导入成功"))
}

// HostsCheck 检测主机
// @Summary 检测主机
// @Description 收集指定主机的系统信息
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "主机ID"
// @Success 200 {object} result.Result "检测完成"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts/{id}/check [post]
func (h *host) HostsCheck(c *gin.Context) {
	id := c.Param("id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		Response(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	err = service.Host.CollectHostInfo(uint(intID))
	if err != nil {
		Response(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, (&result.Result{}).Ok(msg.SUCCSE, nil, "检测完成"))
}

// SyncCloud 同步云主机
// @Summary 同步云主机
// @Description 从云服务商同步主机信息
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param request body SyncCloudRequest true "同步配置"
// @Success 200 {object} result.Result "同步成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "服务器内部错误"
// @Router /api/v1/hosts/sync [post]
func (h *host) HostsSyncCloud(c *gin.Context) {
	var req SyncCloudRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusInternalServerError, (&result.Result{}).Ok(msg.ERROR, nil, "参数验证失败"))
		return
	}

	config := map[string]string{
		"accessKey":    req.AccessKey,
		"accessSecret": req.AccessSecret,
		"regions":      strings.Join(req.Regions, ","),
	}

	if err := service.Host.SyncCloudHosts(req.Provider, config, req.HostGroupId); err != nil {
		c.JSON(http.StatusInternalServerError, (&result.Result{}).Ok(msg.ERROR, nil, err.Error()))
		return
	}

	c.JSON(http.StatusOK, (&result.Result{}).Ok(msg.SUCCSE, nil, "同步成功"))
}
