package ssh

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"strings"
	"sync"
)

func NewSSHClient(conf *SSHClientConfig) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		Timeout:         conf.Timeout,
		User:            conf.UserName,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //忽略know_hosts检查
	}
	switch strings.ToUpper(conf.AuthModel) {
	case "PASSWORD":
		config.Auth = []ssh.AuthMethod{ssh.Password(conf.Password)}
	case "PUBLICKEY":
		signer, err := ssh.ParsePrivateKey([]byte(conf.PublicKey))
		if err != nil {
			return nil, err
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	default:
		fmt.Println("AuthModel is not supported")
	}
	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", conf.IP, conf.Port), config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Turn struct 添加互斥锁以保护并发写入
type Turn struct {
	StdinPipe io.WriteCloser
	Session   *ssh.Session
	WsConn    *websocket.Conn
	mu        sync.Mutex // 添加互斥锁
}

// SetWindowSize 设置终端窗口大小
func (t *Turn) SetWindowSize(rows, cols int) error {
	if t.Session == nil {
		return errors.New("ssh session is nil")
	}
	// 使用 ssh.Window 来设置窗口大小
	return t.Session.WindowChange(rows, cols)
}

func NewTurn(wsConn *websocket.Conn, sshClient *ssh.Client, rows, cols int) (*Turn, error) {
	sess, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return nil, err
	}
	// 创建 Turn 实例
	turn := &Turn{
		StdinPipe: stdinPipe,
		Session:   sess,
		WsConn:    wsConn,
	}
	// 设置标准输出和错误输出
	sess.Stdout = turn
	sess.Stderr = turn
	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// 使用传入的 rows 和 cols 参数请求伪终端
	if err := sess.RequestPty("xterm", rows, cols, modes); err != nil {
		sess.Close()
		return nil, err
	}
	// 启动 shell
	if err := sess.Shell(); err != nil {
		sess.Close()
		return nil, err
	}
	return turn, nil
}

// Write 方法优化
func (t *Turn) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	writer, err := t.WsConn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer writer.Close()
	return writer.Write(p)
}

// Close 方法优化
func (t *Turn) Close() error {
	var closeErr error
	if t.Session != nil {
		if err := t.Session.Close(); err != nil {
			closeErr = err
		}
	}
	if err := t.WsConn.Close(); err != nil && closeErr == nil {
		closeErr = err
	}
	return closeErr
}

// Read 方法优化
func (t *Turn) Read(p []byte) (n int, err error) {
	for {
		msgType, reader, err := t.WsConn.NextReader()
		if err != nil {
			return 0, err
		}
		if msgType != websocket.TextMessage {
			continue
		}
		return reader.Read(p)
	}
}

// LoopRead 优化数据处理
func (t *Turn) LoopRead(context context.Context) error {
	for {
		select {
		case <-context.Done():
			return errors.New("LoopRead exit")
		default:
			_, wsData, err := t.WsConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					return fmt.Errorf("reading webSocket message err: %s", err)
				}
				return nil // 正常关闭
			}
			// 处理输入数据
			var body []byte
			if len(wsData) > 0 {
				if wsData[0] == 0x1 { // 检查是否需要base64解码
					body = decode(wsData[1:])
				} else {
					body = wsData
				}
			}
			if len(body) > 0 {
				if _, err := t.StdinPipe.Write(body); err != nil {
					return fmt.Errorf("StdinPipe write err: %s", err)
				}
			}
		}
	}
}

func (t *Turn) SessionWait() error {
	if err := t.Session.Wait(); err != nil {
		return err
	}
	return nil
}

// decode 函数优化，添加错误处理
func decode(p []byte) []byte {
	decoded, err := base64.StdEncoding.DecodeString(string(p))
	if err != nil {
		// 如果解码失败，返回原始数据
		return p
	}
	return decoded
}
