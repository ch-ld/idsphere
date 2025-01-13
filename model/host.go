package model

import (
	"gorm.io/gorm"
	"time"
)

type HostStatus string
type HostSource string
type SSHAuthType string

const (
	HostStatusOnline  HostStatus = "online"
	HostStatusOffline HostStatus = "offline"
	HostStatusUnknown HostStatus = "unknown"

	HostSourceManual HostSource = "manual"
	HostSourceImport HostSource = "import"
	HostSourceAliyun HostSource = "aliyun"
	HostSourceAWS    HostSource = "aws"

	SSHAuthPassword SSHAuthType = "password"
	SSHAuthKey      SSHAuthType = "publickey"
)

// Host 表示系统中的服务器或机器。
type Host struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index"`
	// HostGroupID 是该主机所属组的ID
	HostGroupID uint `gorm:"not null;index" json:"hostGroupId"`
	// Hostname 是主机的唯一标识符
	Hostname string `gorm:"type:varchar(100);not null;" json:"hostname"`
	// PrivateIP 是主机的内部IP地址
	PrivateIP string `gorm:"type:varchar(15)" json:"privateIp"`
	// PublicIP 是主机的外部IP地址
	PublicIP string `gorm:"type:varchar(15)" json:"publicIp"`
	// SSHPort 是用于SSH连接的端口，默认为22
	SSHPort int `gorm:"default:22" json:"sshPort"`
	// SSHUser 是用于SSH认证的用户名
	SSHUser string `gorm:"type:varchar(50)" json:"sshUser"`
	// SSHAuthType 指定SSH认证类型（密码或密钥）
	SSHAuthType SSHAuthType `gorm:"type:varchar(20);default:'password'" json:"sshAuthType"`
	// SSHPassword 存储SSH认证的密码
	SSHPassword string `gorm:"type:varchar(1024)" json:"sshPassword"`
	// SSHKey 存储SSH认证的密钥
	SSHKey string `gorm:"type:text" json:"sshKey"`
	// CPU CPU核心数
	CPU int `gorm:"comment:'CPU核心数'" json:"cpu"`
	// CPUUsage 当前CPU使用率（百分比）
	CPUUsage float64 `gorm:"comment:'CPU使用率'" json:"cpuUsage"`
	// Memory 总内存大小（GB）
	Memory int `gorm:"comment:'内存大小(GB)'" json:"memory"`
	// MemoryUsage 当前内存使用率（百分比）
	MemoryUsage float64 `gorm:"comment:'内存使用率'" json:"memoryUsage"`
	// DiskSize 总磁盘大小（GB）
	DiskSize int `gorm:"comment:'磁盘大小(GB)'" json:"diskSize"`
	// DiskUsage 当前磁盘使用率（百分比）
	DiskUsage float64 `gorm:"comment:'磁盘使用率'" json:"diskUsage"`
	// OSType 指定操作系统类型
	OSType string `gorm:"type:varchar(50)" json:"osType"`
	// OSVersion 指定操作系统版本
	OSVersion string `gorm:"type:varchar(50)" json:"osVersion"`
	// KernelVersion 指定内核版本
	KernelVersion string `gorm:"type:varchar(50)" json:"kernelVersion"`
	// Status 主机当前的运行状态
	Status HostStatus `gorm:"type:varchar(20);default:'unknown'" json:"status"`
	// Source 主机是如何被添加到系统中的
	Source HostSource `gorm:"type:varchar(20);not null" json:"source"`
	// CloudInstanceID 云实例的唯一标识符
	//CloudInstanceID *string `gorm:"type:varchar(100);unique" json:"cloudInstanceId"`
	CloudInstanceID *string `gorm:"type:varchar(255);unique" json:"cloudInstanceId"`
	// Region 指定主机所在的地理区域
	Region string `gorm:"type:varchar(50)" json:"region"`
	// Tags 主机的标签，用于分类和搜索
	Tags string `gorm:"type:text" json:"tags"`
	// Description 提供关于主机的额外描述信息
	Description string `gorm:"type:varchar(255)" json:"description"`
	// LastCheckTime 记录主机最后一次被检查或更新的时间
	LastCheckTime *time.Time `json:"lastCheckTime"`
	// HostGroup 表示该主机所属的组
	HostGroup HostGroup `gorm:"foreignKey:HostGroupID" json:"hostGroup"`
}
