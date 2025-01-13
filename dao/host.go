package dao

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"ops-api/global"
	"ops-api/model"
)

var Host host

type host struct{}

// HostCreate 创建主机
func (h *host) HostCreate(host *model.Host) error {
	return global.MySQLClient.Transaction(func(tx *gorm.DB) error {
		// 1. 创建主机
		if err := tx.Preload("HostGroup").Create(host).Error; err != nil {
			return err
		}
		// 2. 更新主机组计数
		if err := tx.Model(&model.HostGroup{}).
			Where("id = ?", host.HostGroupID).
			Update("host_count", gorm.Expr("host_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})
}

// HostsBatchCreate 批量创建主机
func (h *host) HostsBatchCreate(hosts []*model.Host) error {
	if len(hosts) == 0 {
		return fmt.Errorf("没有要导入的主机数据")
	}

	return global.MySQLClient.Transaction(func(tx *gorm.DB) error {
		// 批量创建前进行验证
		for i, host := range hosts {
			if err := h.ValidateHost(host, i+1); err != nil {
				return err
			}
		}
		// 使用循环逐个创建，以便更好地处理错误
		for i, host := range hosts {
			if err := tx.Preload("HostGroup").Create(host).Error; err != nil {
				return fmt.Errorf("创建第 %d 条记录失败: %v", i+1, err)
			}
		}
		// 2. 统计并更新每个主机组的计数
		groupCounts := make(map[uint]int64)
		for _, host := range hosts {
			groupCounts[host.HostGroupID]++
		}
		// 3. 更新主机组计数
		for groupID, count := range groupCounts {
			if err := tx.Model(&model.HostGroup{}).
				Where("id = ?", groupID).
				Update("host_count", gorm.Expr("host_count + ?", count)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// 查询主机列表
func (h *host) HostsList(params map[string]interface{}, offset, limit int) ([]model.Host, int64, error) {
	var hosts []model.Host
	var total int64

	query := global.MySQLClient.Model(&model.Host{})

	// 条件查询
	if groupID, ok := params["hostGroupId"].(uint); ok && groupID > 0 {
		query = query.Where("host_group_id = ?", groupID)
	}
	if keyword, ok := params["keyword"].(string); ok && keyword != "" {
		query = query.Where("hostname LIKE ? OR private_ip LIKE ? OR public_ip LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if region, ok := params["region"].(string); ok && region != "" {
		query = query.Where("region = ?", region)
	}
	if source, ok := params["source"].(string); ok && source != "" {
		query = query.Where("source =?", source)
	}
	if osVersion, ok := params["osVersion"].(string); ok && osVersion != "" {
		query = query.Where("os_version =?", osVersion)
	}
	if status, ok := params["status"].(string); ok && status != "" {
		query = query.Where("status =?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("HostGroup").Offset(offset).Limit(limit).Find(&hosts).Error
	return hosts, total, err
}

// 根据主机ID查询主机
func (h *host) HostGetByID(id uint) (*model.Host, error) {
	var host model.Host
	if err := global.MySQLClient.Preload("HostGroup").First(&host, id).Error; err != nil {
		return nil, err
	}
	return &host, nil
}

// HostUpdate 更新主机信息
func (h *host) HostUpdate(id uint, updates *model.Host) error {
	return global.MySQLClient.Preload("HostGroup").Model(&model.Host{}).Where("id = ?", id).Updates(updates).Error
}

// HostDelete 删除主机
func (h *host) HostDelete(id uint) error {
	return global.MySQLClient.Transaction(func(tx *gorm.DB) error {
		// 1. 先查询主机，获取HostGroupID
		var host model.Host
		if err := tx.Preload("HostGroup").First(&host, id).Error; err != nil {
			return err
		}
		// 记录HostGroupID
		hostGroupID := host.HostGroupID
		// 2. 删除主机
		if err := tx.Preload("HostGroup").Delete(&host).Error; err != nil {
			return err
		}
		// 3. 查询该主机组剩余主机数量
		var count int64
		err := tx.Preload("HostGroup").Model(&model.Host{}).Where("host_group_id = ?", hostGroupID).Count(&count).Error
		if err != nil {
			return err
		}

		// 4. 更新主机组的host_count
		return tx.Model(&model.HostGroup{}).
			Where("id = ?", hostGroupID).
			Update("host_count", count).Error
	})
}

// HostsBatchDelete 批量删除主机
func (h *host) HostsBatchDelete(ids []uint) error {
	return global.MySQLClient.Transaction(func(tx *gorm.DB) error {
		// 1. 查询要删除的主机，获取涉及的主机组
		var hosts []model.Host
		if err := tx.Preload("HostGroup").Find(&hosts, ids).Error; err != nil {
			return err
		}

		// 2. 获取涉及的主机组ID
		groupIDs := make(map[uint]bool)
		for _, host := range hosts {
			groupIDs[host.HostGroupID] = true
		}

		// 3. 删除主机
		if err := tx.Preload("HostGroup").Delete(&model.Host{}, ids).Error; err != nil {
			return err
		}

		// 4. 更新每个受影响的主机组计数
		for groupID := range groupIDs {
			var count int64
			err := tx.Model(&model.Host{}).Where("host_group_id = ?", groupID).Count(&count).Error
			if err != nil {
				return err
			}

			if err := tx.Model(&model.HostGroup{}).
				Where("id = ?", groupID).
				Update("host_count", count).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// IsHostnameExists 检查主机名是否已存在
func (h *host) IsHostnameExists(hostname string) (bool, uint) {
	var existingHost model.Host
	if err := global.MySQLClient.Where("hostname = ?", hostname).First(&existingHost).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0
		} else {
			// 其他错误情况
			return false, 0
		}
	}
	// 找到主机
	return true, existingHost.ID
}

// ValidateHost 验证主机数据
func (h *host) ValidateHost(host *model.Host, lineNum int) error {
	if host.HostGroupID == 0 {
		return fmt.Errorf("第 %d 行主机组ID无效", lineNum)
	}
	if host.Hostname == "" {
		return fmt.Errorf("第 %d 行主机名不能为空", lineNum)
	}
	if host.PrivateIP == "" {
		return fmt.Errorf("第 %d 行私有IP不能为空", lineNum)
	}
	if host.SSHPort <= 0 || host.SSHPort > 65535 {
		return fmt.Errorf("第 %d 行SSH端口范围无效", lineNum)
	}
	return nil
}
