package service

import (
	"errors"
	"ops-api/dao"
	"ops-api/model"
)

var HostGroup hostGroup

type hostGroup struct{}

// HostGroupInput 创建主机组输入参数
type HostGroupInput struct {
	Name        string `json:"name" binding:"required,min=2,max=50"`
	Description string `json:"description" binding:"max=255"`
}

// 创建主机组
func (hg *hostGroup) Create(input *HostGroupInput) error {
	hostGroup := &model.HostGroup{
		Name:        input.Name,
		Description: input.Description,
	}
	return dao.HostGroup.Create(hostGroup)
}

// 更新主机组
func (hg *hostGroup) Update(id uint, input *HostGroupInput) error {
	updates := map[string]interface{}{
		"name":        input.Name,
		"description": input.Description,
	}
	return dao.HostGroup.Update(id, updates)
}

// 删除主机组
func (hg *hostGroup) Delete(id uint) error {
	// 检查是否存在关联的主机
	group, err := dao.HostGroup.GetByID(id)
	if err != nil {
		return err
	}
	if group.HostCount > 0 {
		return errors.New("该主机组下还有主机，无法删除")
	}
	return dao.HostGroup.Delete(id)
}

// 查询主机组列表
func (hg *hostGroup) List(name string, page, limit int) (hostGroups []model.HostGroup, total int64, err error) {
	return dao.HostGroup.List(name, page, limit)
}

// 根据ID查询主机组
func (hg *hostGroup) GetByID(id uint) (*model.HostGroup, error) {
	return dao.HostGroup.GetByID(id)
}

// 根据名称查询主机组
func (hg *hostGroup) GetByName(name string) (uint, error) {
	return dao.HostGroup.GetByName(name)
}

// 批量删除主机组
func (hg *hostGroup) BatchDelete(ids []uint) error {
	return dao.HostGroup.BatchDelete(ids)
}
