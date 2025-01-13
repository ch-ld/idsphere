package routers

import (
	"github.com/gin-gonic/gin"
	"ops-api/controller"
)

func initSftpRouters(router *gin.Engine) {
	// sftp列出目录信息
	router.GET("/api/v1/sftp/list", controller.Sftp.SftpListDirectory)
	// sftp上传文件
	router.POST("/api/v1/sftp/upload", controller.Sftp.SftpUploadFile)
	// sftp下载文件
	router.GET("/api/v1/sftp/download", controller.Sftp.SftpDownloadFile)
	// sftp创建目录
	router.POST("/api/v1/sftp/mkdir", controller.Sftp.SftpCreateDirectory)
	// sftp删除路径
	router.POST("/api/v1/sftp/delete", controller.Sftp.SftpDeletePath)
}
