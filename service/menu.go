package service

import (
	"ops-api/dao"
	"ops-api/model"
)

var Menu menu

type menu struct{}

// GetMenuListAll 获取菜单列表（权限分配）
func (m *menu) GetMenuListAll() (data *dao.MenuList, err error) {
	data, err = dao.Menu.GetMenuListAll()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetMenuList 获取菜单列表
func (m *menu) GetMenuList(title string, page, limit int) (data *dao.MenuList, err error) {
	data, err = dao.Menu.GetMenuList(title, page, limit)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// CreateMenu 创建菜单
func (m *menu) CreateMenu(menu *model.Menu) error {
	return dao.Menu.CreateMenu(menu)
}

// CreateSubMenu 创建子菜单
func (m *menu) CreateSubMenu(subMenu *model.SubMenu) error {
	return dao.Menu.CreateSubMenu(subMenu)
}

// UpdateMenu 更新菜单
func (m *menu) UpdateMenu(id uint, updates map[string]interface{}) error {
	return dao.Menu.UpdateMenu(id, updates)
}

// UpdateSubMenu 更新子菜单
func (m *menu) UpdateSubMenu(id uint, updates map[string]interface{}) error {
	return dao.Menu.UpdateSubMenu(id, updates)
}

// DeleteMenu 删除菜单
func (m *menu) DeleteMenu(id uint) error {
	return dao.Menu.DeleteMenu(id)
}

// DeleteSubMenu 删除子菜单
func (m *menu) DeleteSubMenu(id uint) error {
	return dao.Menu.DeleteSubMenu(id)
}

// UpdateMenuSort 更新菜单排序
func (m *menu) UpdateMenuSort(id uint, sort int) error {
	return dao.Menu.UpdateMenuSort(id, sort)
}

// UpdateSubMenuSort 更新子菜单排序
func (m *menu) UpdateSubMenuSort(id uint, sort int) error {
	return dao.Menu.UpdateSubMenuSort(id, sort)
}
