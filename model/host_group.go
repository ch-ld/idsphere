package model

import "gorm.io/gorm"

// 主机组
type HostGroup struct {
	gorm.Model
	Name        string `gorm:"type:varchar(50);not null;comment:'主机组名称'" json:"name"`
	Description string `gorm:"type:varchar(255);comment:'描述'" json:"description"`
	HostCount   int    `gorm:"default:0;comment:'主机数量'" json:"hostCount"`
}

// 表名
func (*HostGroup) TableName() (name string) {
	return "host_group"
}
