package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"ops-api/service"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var Sftp sftp

type sftp struct{}

// SftpListDirectory 获取 SFTP 目录
// @Summary 获取SFTP目录列表
// @Description 获取指定主机上SFTP的目录文件列表
// @Tags SFTP管理
// @Accept json
// @Produce json
// @Param hostId query string true "主机ID"
// @Param path query string false "目录路径，默认为/tmp/"
// @Success 200 {object} result.Result "成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "内部服务器错误"
// @Router /api/v1/sftp/list [get]
func (s *sftp) SftpListDirectory(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	sortBy := c.DefaultQuery("sortBy", "name")      // 排序字段
	sortOrder := c.DefaultQuery("sortOrder", "asc") // 排序方向
	path := c.Query("path")
	hostID := c.Query("hostId")

	if hostID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostId不能为空"})
		return
	}

	UintHostID, err := strconv.ParseUint(hostID, 10, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id转换错误"})
		return
	}

	// 获取所有文件
	files, path, err := service.Sftp.SftpListDirectory(uint(UintHostID), path)
	if err != nil {
		Response(c, http.StatusInternalServerError, err.Error())
		return
	}
	// 先按文件夹和文件分组排序
	sort.SliceStable(files, func(i, j int) bool {
		// 如果一个是目录一个是文件，目录排在前面
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}

		// 根据选择的排序字段排序
		switch sortBy {
		case "name":
			if sortOrder == "desc" {
				return strings.ToLower(files[i].Name) > strings.ToLower(files[j].Name)
			}
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		case "size":
			if sortOrder == "desc" {
				return files[i].Size > files[j].Size
			}
			return files[i].Size < files[j].Size
		case "modTime":
			if sortOrder == "desc" {
				return files[i].ModTime.After(files[j].ModTime)
			}
			return files[i].ModTime.Before(files[j].ModTime)
		default:
			// 默认按名称排序
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		}
	})

	// 计算总数
	total := len(files)

	// 计算分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// 返回分页后的数据
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": files[start:end],
		"path": path,
		"pagination": gin.H{
			"current":  page,
			"pageSize": pageSize,
			"total":    total,
		},
	})
}

// SftpUploadFile godoc
// @Summary 上传文件到SFTP
// @Description 通过SFTP上传文件到指定主机的指定路径
// @Tags SFTP管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "要上传的文件"
// @Param hostId formData string true "主机ID"
// @Param path formData string true "目标路径"
// @Success 200 {object} result.Result{data=string} "上传成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "内部服务器错误"
// @Router /api/v1/upload [post]
// SftpUploadFile 上传文件
func (s *sftp) SftpUploadFile(c *gin.Context) {
	// 获取表单参数
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取上传文件失败: " + err.Error()})
		return
	}
	defer file.Close()
	// 获取目标路径参数
	remotePath := c.PostForm("path")
	if remotePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标路径不能为空"})
		return
	}
	hostID := c.PostForm("hostId")
	if hostID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostId不能为空"})
		return
	}
	UintHostID, err := strconv.ParseUint(hostID, 10, 0)
	if err != nil {
		fmt.Println("转换错误:", err)
		return
	}
	err = service.Sftp.SftpUploadFile(uint(UintHostID), remotePath, file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "文件上传成功",
		"data": remotePath,
	})
}

// SftpDownloadFile godoc
// @Summary 从SFTP下载文件
// @Description 从指定主机下载SFTP文件
// @Tags SFTP管理
// @Accept json
// @Produce octet-stream
// @Param hostId query string true "主机ID"
// @Param path query string true "文件路径"
// @Success 200 {file} binary "文件内容"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "内部服务器错误"
// @Router /api/v1/download [get]
// SftpDownloadFile 下载文件处理
func (s *sftp) SftpDownloadFile(c *gin.Context) {
	// 参数验证
	hostID := c.Query("hostId")
	if hostID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostId不能为空"})
		return
	}

	UintHostID, err := strconv.ParseUint(hostID, 10, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostId格式错误"})
		return
	}

	remotePath := c.Query("path")
	if remotePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不能为空"})
		return
	}

	// 调用service层下载文件
	fileInfo, reader, err := service.Sftp.SftpDownloadFile(uint(UintHostID), remotePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	// 获取文件名
	fileName := filepath.Base(remotePath)

	// 设置响应头
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 发送文件内容
	if _, err := io.Copy(c.Writer, reader); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件传输失败"})
		return
	}
}

// CreateDirectory 创建目录
// @Summary 创建SFTP目录
// @Description 在指定主机上创建SFTP目录
// @Tags SFTP管理
// @Accept json
// @Produce json
// @Param request body service.SftpCmd true "创建目录请求参数"
// @Success 200 {object} result.Result{data=string} "创建成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "内部服务器错误"
// @Router /api/v1/sftp/mkdir [post]
func (s *sftp) SftpCreateDirectory(c *gin.Context) {
	var r service.SftpCmd
	if err := c.ShouldBind(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := service.Sftp.SftpCreateDirectory(r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "目录创建成功",
		"data": r.Path,
	})
}

// DeletePath 删除文件或目录
// @Summary 删除SFTP文件或目录
// @Description 删除指定主机上的SFTP文件或目录
// @Tags SFTP管理
// @Accept json
// @Produce json
// @Param request body service.SftpCmd true "删除请求参数"
// @Success 200 {object} result.Result{data=string} "删除成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "内部服务器错误"
// @Router /api/v1/sftp/delete [post]
func (s *sftp) SftpDeletePath(c *gin.Context) {
	var r service.SftpCmd
	if err := c.ShouldBind(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := service.Sftp.SftpDeletePath(r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "文件删除成功",
		"data": r.Path,
	})
}
