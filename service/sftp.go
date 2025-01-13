package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"ops-api/dao"
	"ops-api/utils"
	ssh1 "ops-api/utils/ssh"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Sftp sftp

type sftp struct{}

type SftpCmd struct {
	HostId int    `json:"hostId"`
	Path   string `json:"path"`
}

type SshCmd struct {
	HostId  int    `json:"hostId"`
	Command string `json:"command"`
}

// Sftp 列出目录以及文件信息
func (s *sftp) SftpListDirectory(hostID uint, path string) ([]ssh1.FileInfo, string, error) {
	// 获取SSH配置
	config, err := s.SSHConnect(hostID)
	if path == "" {
		path = "/tmp/"
	}
	// 创建SFTP客户端
	sftpClient, err := ssh1.NewSFTPClient(config)
	if err != nil {
		return nil, "", fmt.Errorf("ssh创建客户端失败")
	}
	defer sftpClient.Close()

	// 获取目录列表
	files, err := sftpClient.ListDir(path)
	if err != nil {
		return nil, "", fmt.Errorf("sftp获取目录列表失败")
	}

	return files, path, nil
}

// Sftp 上传
func (s *sftp) SftpUploadFile(hostID uint, path string, file multipart.File, header *multipart.FileHeader) error {
	// 获取SSH配置
	config, err := s.SSHConnect(hostID)
	if path == "" {
		path = "/tmp/"
	}
	// 创建临时文件
	tempFile := filepath.Join(os.TempDir(), header.Filename)
	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("sftp创建临时文件失败")
	}
	defer os.Remove(tempFile)
	defer out.Close()

	// 保存文件
	_, err = io.Copy(out, file)
	if err != nil {
		return fmt.Errorf("sftp保存文件失败")
	}

	// 创建SFTP客户端并上传文件
	sftpClient, err := ssh1.NewSFTPClient(config)
	if err != nil {
		return fmt.Errorf("创建sftp客户端失败")
	}
	defer sftpClient.Close()

	remoteFilePath := filepath.Join(path, header.Filename)
	err, _ = sftpClient.UploadFile(tempFile, remoteFilePath)
	if err != nil {
		return fmt.Errorf("sftp上传文件失败")
	}
	return nil
}

// SftpDownloadFile 下载文件
func (s *sftp) SftpDownloadFile(hostID uint, path string) (os.FileInfo, io.ReadCloser, error) {
	// 获取SSH配置
	config, err := s.SSHConnect(hostID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取SSH配置失败: %v", err)
	}

	// 创建SFTP客户端
	sftpClient, err := ssh1.NewSFTPClient(config)
	if err != nil {
		return nil, nil, fmt.Errorf("创建SFTP客户端失败: %v", err)
	}

	// 打开远程文件
	remoteFile, err := sftpClient.Open(path)
	if err != nil {
		sftpClient.Close()
		return nil, nil, fmt.Errorf("打开远程文件失败: %v", err)
	}

	// 获取文件信息
	fileInfo, err := remoteFile.Stat()
	if err != nil {
		remoteFile.Close()
		sftpClient.Close()
		return nil, nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 创建一个读取器包装器，用于在读取完成后清理资源
	reader := &sftpReadCloser{
		Reader:     remoteFile,
		remoteFile: remoteFile,
		sftpClient: sftpClient,
	}

	return fileInfo, reader, nil
}

// CreateDirectory 创建目录
func (s *sftp) SftpCreateDirectory(r SftpCmd) error {
	// 获取SSH配置
	config, err := s.SSHConnect(uint(r.HostId))
	// 创建SFTP客户端
	sftpClient, err := ssh1.NewSFTPClient(config)
	if err != nil {
		return fmt.Errorf("创建sftp客户端失败")
	}
	defer sftpClient.Close()
	// 创建目录
	err = sftpClient.MakeDir(r.Path)
	if err != nil {
		return fmt.Errorf("创建目录失败")
	}
	return nil
}

// DeletePath 删除文件或目录
func (s *sftp) SftpDeletePath(r SftpCmd) error {
	// 获取SSH配置
	config, err := s.SSHConnect(uint(r.HostId))
	// 创建SFTP客户端
	sftpClient, err := ssh1.NewSFTPClient(config)
	if err != nil {
		return fmt.Errorf("创建sftp客户端失败")
	}
	defer sftpClient.Close()
	// 获取文件信息
	fileInfo, err := sftpClient.Stat(r.Path)
	if err != nil {
		return fmt.Errorf("获取文件信息失败")
	}
	// 如果是目录，递归删除
	if fileInfo.IsDir() {
		err = sftpClient.RemoveDirectory(r.Path)
	} else {
		err = sftpClient.Remove(r.Path)
	}
	return nil
}

// sftpReadCloser 包装器结构体，用于确保资源正确释放
type sftpReadCloser struct {
	io.Reader
	remoteFile io.ReadCloser
	sftpClient *ssh1.SFTPClient
}

// Close 实现io.ReadCloser接口
func (s *sftpReadCloser) Close() error {
	// 先关闭文件
	fileErr := s.remoteFile.Close()
	// 再关闭SFTP客户端
	s.sftpClient.Close() // 即使这个方法没有返回值，我们也执行它

	// 返回文件关闭时的错误（如果有的话）
	return fileErr
}

// SSHConnect 用于创建SSH/SFTP连接的通用方法
func (s *sftp) SSHConnect(hostID uint) (*ssh1.SSHClientConfig, error) {
	// 获取主机信息
	host, err := dao.Host.HostGetByID(hostID)
	if err != nil {
		return nil, fmt.Errorf("获取主机信息失败: %v", err)
	}

	// 基础配置
	config := &ssh1.SSHClientConfig{
		Timeout:  time.Second * 5,
		IP:       host.PrivateIP,
		Port:     host.SSHPort,
		UserName: host.SSHUser,
	}

	// 根据认证类型设置认证信息
	switch strings.ToUpper(string(host.SSHAuthType)) {
	case "PASSWORD":
		password, err := utils.Decrypt(host.SSHPassword)
		if err != nil {
			return nil, fmt.Errorf("解密主机SSH密码失败: %v", err)
		}
		config.Password = password
		config.AuthModel = "PASSWORD"
	case "PUBLICKEY":
		privateKey, err := utils.Decrypt(host.SSHKey)
		if err != nil {
			return nil, fmt.Errorf("解密主机SSH私钥失败: %v", err)
		}
		config.PublicKey = privateKey
		config.AuthModel = "PUBLICKEY"
	default:
		return nil, fmt.Errorf("不支持的SSH认证类型: %s", host.SSHAuthType)
	}

	return config, nil
}
