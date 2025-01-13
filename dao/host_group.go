package dao

import (
	"ops-api/global"
	"ops-api/model"
)

var HostGroup hostGroup

type hostGroup struct{}

// 查询主机列表
func (hg *hostGroup) List(name string, page, limit int) (hostGroups []model.HostGroup, total int64, err error) {
	// 定义数据的起始位置
	startSet := (page - 1) * limit

	// 获取主机组列表
	tx := global.MySQLClient.Model(&model.HostGroup{}).
		Where("name LIKE ?", "%"+name+"%").
		Count(&total). // 获取总数
		Limit(limit).
		Offset(startSet).
		Find(&hostGroups)
	if tx.Error != nil {
		return nil, 0, err
	}

	return hostGroups, total, nil
}

// 创建主机组
func (hg *hostGroup) Create(hostGroup *model.HostGroup) error {
	return global.MySQLClient.Create(hostGroup).Error
}

// 更新主机组
func (hg *hostGroup) Update(id uint, hostGroup map[string]interface{}) error {
	return global.MySQLClient.Model(&model.HostGroup{}).Where("id = ?", id).Updates(hostGroup).Error
}

// 删除主机组
func (hg *hostGroup) Delete(id uint) error {
	return global.MySQLClient.Delete(&model.HostGroup{}, id).Error
}

// 根据ID查询主机组
func (hg *hostGroup) GetByID(id uint) (*model.HostGroup, error) {
	var group model.HostGroup
	err := global.MySQLClient.First(&group, id).Error
	return &group, err
}

// 根据名称查询主机组
func (hg *hostGroup) GetByName(name string) (uint, error) {
	var group model.HostGroup
	err := global.MySQLClient.Where("name = ?", name).First(&group).Error
	return group.ID, err
}

// 批量删除主机组
func (hg *hostGroup) BatchDelete(ids []uint) error {
	return global.MySQLClient.Delete(&model.HostGroup{}, ids).Error
}
