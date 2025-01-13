package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"ops-api/service"
	"ops-api/utils"
	"ops-api/utils/msg"
	"ops-api/utils/result"
	ssh1 "ops-api/utils/ssh"
	"strconv"
	"strings"
	"time"
)

var SSH ssh

type ssh struct{}

// WebSSHConfig WebSSH配置结构
type WebSSHConfig struct {
	IP        string `json:"ip" form:"ip"`
	Port      string `json:"port" form:"port"`
	Username  string `json:"username" form:"username"`
	Password  string `json:"password" form:"password"`
	PublicKey string `json:"public_key" form:"public_key"`
	AuthModel string `json:"authmodel" form:"authmodel"`
	Cols      int    `json:"cols" form:"cols"`
	Rows      int    `json:"rows" form:"rows"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024, // 增加缓冲区大小
	WriteBufferSize: 32 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ssh 执行远程命令
// @Summary 执行SSH远程命令
// @Description 在指定主机上执行SSH远程命令
// @Tags SSH管理
// @Accept json
// @Produce json
// @Param request body service.SshCmd true "SSH命令请求参数"
// @Success 200 {object} result.Result{data=string} "执行成功"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "内部服务器错误"
// @Router /cmdb/ssh/command [post]
func (s *ssh) SshCommand(c *gin.Context) {
	var cmd service.SshCmd
	// 绑定并校验请求参数
	if err := c.ShouldBind(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": err.Error(),
		})
		return
	}
	output, err := service.SSH.SshCommand(cmd)
	if err != nil {
		c.JSON(http.StatusOK, (&result.Result{}).Error(msg.ERROR, err.Error(), msg.GetErrMsg(msg.ERROR)))
		return
	}
	formattedOutput := strings.ReplaceAll(output, "\n", "<br>")
	c.JSON(http.StatusOK, (&result.Result{}).Ok(200, formattedOutput, msg.GetErrMsg(200)))
}

// webssh终端
// @Summary WebSSH终端连接
// @Description 建立WebSocket连接以提供Web终端功能
// @Tags SSH管理
// @Accept json
// @Produce json
// @Param ip query string true "主机IP地址"
// @Param port query string true "SSH端口"
// @Param username query string true "用户名"
// @Param password query string true "加密后的密码"
// @Param authmodel query string true "认证模式" Enums(password, key)
// @Param cols query int false "终端列数" default(80)
// @Param rows query int false "终端行数" default(24)
// @Success 101 {string} string "Switching Protocols to websocket"
// @Failure 400 {object} result.Result "请求参数错误"
// @Failure 500 {object} result.Result "内部服务器错误"
// @Router /cmdb/ssh/webssh [get]
func (s *ssh) WebSsh(c *gin.Context) {
	var config WebSSHConfig
	if err := c.ShouldBindQuery(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	// 参数验证
	if config.IP == "" || config.Port == "" || config.Username == "" || config.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "必要参数不能为空"})
		return
	}

	// 设置默认终端大小
	if config.Cols == 0 {
		config.Cols = 80
	}
	if config.Rows == 0 {
		config.Rows = 24
	}

	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer wsConn.Close()

	// 设置WebSocket读写超时
	wsConn.SetReadDeadline(time.Time{})  // 禁用读超时
	wsConn.SetWriteDeadline(time.Time{}) // 禁用写超时

	// 创建SSH客户端配置
	port, _ := strconv.Atoi(config.Port)
	sshConfig := &ssh1.SSHClientConfig{
		Timeout:  time.Second * 30, // 增加超时时间
		IP:       config.IP,
		Port:     port,
		UserName: config.Username,
	}
	// 处理认证
	switch strings.ToUpper(config.AuthModel) {
	case "PASSWORD":
		password, err := utils.Decrypt(config.Password)
		if err != nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte("解密密码失败: "+err.Error()))
			return
		}
		sshConfig.Password = password
		sshConfig.AuthModel = "PASSWORD"
	case "PUBLICKEY":
		privateKey, err := utils.Decrypt(config.PublicKey)
		if err != nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte("解密私钥失败: "+err.Error()))
			return
		}
		sshConfig.PublicKey = privateKey
		sshConfig.AuthModel = "PUBLICKEY"
	default:
		wsConn.WriteMessage(websocket.TextMessage, []byte("不支持的SSH认证类型"))
		return
	}

	// 建立SSH连接
	sshClient, err := ssh1.NewSSHClient(sshConfig)
	if err != nil {
		log.Printf("Failed to create SSH client: %v", err)
		wsConn.WriteMessage(websocket.TextMessage, []byte("SSH连接失败: "+err.Error()))
		return
	}
	defer sshClient.Close()

	// 创建终端
	turn, err := ssh1.NewTurn(wsConn, sshClient, config.Rows, config.Cols)
	if err != nil {
		log.Printf("Failed to create terminal: %v", err)
		wsConn.WriteMessage(websocket.TextMessage, []byte("终端创建失败: "+err.Error()))
		return
	}
	defer turn.Close()

	// 设置初始终端大小
	if err := turn.SetWindowSize(config.Rows, config.Cols); err != nil {
		log.Printf("Failed to set window size: %v", err)
	}

	// 创建上下文和错误通道
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errChan := make(chan error, 3) // 增加通道缓冲区

	// 启动数据读取协程
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in read loop: %v", r)
			}
		}()
		errChan <- turn.LoopRead(ctx)
	}()

	// 处理窗口大小调整和数据写入
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in write loop: %v", r)
			}
			cancel()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				messageType, message, err := wsConn.ReadMessage()
				if err != nil {
					if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
						log.Printf("read error: %v", err)
					}
					return
				}

				if messageType == websocket.TextMessage {
					// 处理窗口大小调整消息
					var resizeMessage struct {
						Type string `json:"type"`
						Rows int    `json:"rows"`
						Cols int    `json:"cols"`
					}
					if err := json.Unmarshal(message, &resizeMessage); err == nil && resizeMessage.Type == "resize" {
						if err := turn.SetWindowSize(resizeMessage.Rows, resizeMessage.Cols); err != nil {
							log.Printf("resize error: %v", err)
						}
						continue
					}

					// 处理普通数据
					if len(message) > 0 {
						if _, err := turn.StdinPipe.Write(message); err != nil {
							log.Printf("write error: %v", err)
							return
						}
					}
				}
			}
		}
	}()

	// 等待会话结束
	select {
	case err := <-errChan:
		if err != nil && err != io.EOF {
			log.Printf("Session error: %v", err)
			wsConn.WriteMessage(websocket.TextMessage, []byte("会话错误: "+err.Error()))
		}
	case <-ctx.Done():
		log.Println("Session closed")
	}
}
