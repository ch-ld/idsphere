package service

import (
	"fmt"
	ssh1 "ops-api/utils/ssh"
)

var SSH ssh

type ssh struct{}

// 执行远程命令
func (s *ssh) SshCommand(cmd SshCmd) (string, error) {
	// 获取SSH配置
	config, err := Sftp.SSHConnect(uint(cmd.HostId))
	// 开始处理 Sftp 会话
	output, err := ssh1.SshCommand(config, cmd.Command)
	if err != nil {
		return "", fmt.Errorf("执行远程命令失败")
	}
	return output, nil
}
