package dao

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log"
	"ops-api/global"
	"ops-api/model"
)

var Menu menu

type menu struct{}

// MenuList 返回给前端菜单列表结构体
type MenuList struct {
	Items []*model.Menu `json:"items"`
	Total int64         `json:"total"`
}

// MenuItem 菜单项
type MenuItem struct {
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	Component string            `json:"component"`
	Meta      map[string]string `json:"meta"`
	Redirect  string            `json:"redirect,omitempty"`
	Children  []*MenuItem       `json:"children,omitempty"` // 当Children为Null时不返回，否则前端无法正确加载路由
}

// 添加错误定义
var (
	ErrMenuNameExists   = errors.New("菜单名称已存在")
	ErrMenuPathExists   = errors.New("菜单路径已存在")
	ErrMenuNotFound     = errors.New("菜单不存在")
	ErrSubMenuNotFound  = errors.New("子菜单不存在")
	ErrParentMenuExists = errors.New("父级菜单已存在子菜单，无法删除")
)

// CreateMenu 创建菜单
func (m *menu) CreateMenu(menu *model.Menu) error {
	// 检查菜单名称是否已存在
	var count int64
	if err := global.MySQLClient.Model(&model.Menu{}).
		Where("name = ? OR path = ?", menu.Name, menu.Path).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrMenuNameExists
	}

	return global.MySQLClient.Create(menu).Error
}

// CreateSubMenu 创建子菜单
func (m *menu) CreateSubMenu(subMenu *model.SubMenu) error {
	// 检查父菜单是否存在
	var parentMenu model.Menu
	if err := global.MySQLClient.First(&parentMenu, subMenu.MenuID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMenuNotFound
		}
		return err
	}

	// 检查子菜单名称是否已存在
	var count int64
	if err := global.MySQLClient.Model(&model.SubMenu{}).
		Where("name = ? OR path = ?", subMenu.Name, subMenu.Path).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrMenuNameExists
	}

	return global.MySQLClient.Create(subMenu).Error
}

// GetMenuListAll 获取所有菜单（权限分配）
func (m *menu) GetMenuListAll() (data *MenuList, err error) {

	// 定义返回的内容
	var (
		menus []*model.Menu
		total int64
	)

	// 获取所有菜单
	tx := global.MySQLClient.Model(&model.Menu{}).
		Preload("SubMenus", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort")
		}). // 加载二级菜单
		Count(&total).
		Order("sort").
		Find(&menus)
	if tx.Error != nil {
		return nil, err
	}

	return &MenuList{
		Items: menus,
		Total: total,
	}, nil
}

// UpdateMenu 更新菜单
func (m *menu) UpdateMenu(id uint, updates map[string]interface{}) error {
	// 检查菜单是否存在
	var menu model.Menu
	if err := global.MySQLClient.First(&menu, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMenuNotFound
		}
		return err
	}

	// 如果更新了name或path，需要检查是否与其他菜单冲突
	if name, ok := updates["name"]; ok {
		var count int64
		if err := global.MySQLClient.Model(&model.Menu{}).
			Where("name = ? AND id != ?", name, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrMenuNameExists
		}
	}

	if path, ok := updates["path"]; ok {
		var count int64
		if err := global.MySQLClient.Model(&model.Menu{}).
			Where("path = ? AND id != ?", path, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrMenuPathExists
		}
	}

	return global.MySQLClient.Model(&menu).Updates(updates).Error
}

// UpdateSubMenu 更新子菜单
func (m *menu) UpdateSubMenu(id uint, updates map[string]interface{}) error {
	// 检查子菜单是否存在
	var subMenu model.SubMenu
	//if err := global.MySQLClient.First(&subMenu, id).Error; err != nil {
	//	if errors.Is(err, gorm.ErrRecordNotFound) {
	//		return ErrSubMenuNotFound
	//	}
	//	return err
	//}
	if err := global.MySQLClient.Where("id = ?", id).First(&subMenu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSubMenuNotFound
		}
		return err
	}

	// 如果更新了name或path，需要检查是否与其他子菜单冲突
	if name, ok := updates["name"]; ok {
		var count int64
		if err := global.MySQLClient.Model(&model.SubMenu{}).
			Where("name = ? AND id != ?", name, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrMenuNameExists
		}
	}

	if path, ok := updates["path"]; ok {
		var count int64
		if err := global.MySQLClient.Model(&model.SubMenu{}).
			Where("path = ? AND id != ?", path, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrMenuPathExists
		}
	}

	return global.MySQLClient.Model(&subMenu).Updates(updates).Error
}

// DeleteMenu 删除菜单
func (m *menu) DeleteMenu(id uint) error {
	// 检查菜单是否存在
	var menu model.Menu
	if err := global.MySQLClient.First(&menu, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMenuNotFound
		}
		return err
	}

	// 检查是否有子菜单
	var count int64
	if err := global.MySQLClient.Model(&model.SubMenu{}).
		Where("menu_id = ?", id).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrParentMenuExists
	}
	return global.MySQLClient.Delete(&menu).Error
}

// DeleteSubMenu 删除子菜单
func (m *menu) DeleteSubMenu(id uint) error {
	// 检查子菜单是否存在
	var subMenu model.SubMenu
	if err := global.MySQLClient.Where("id = ?", id).First(&subMenu).Error; err != nil {
		log.Printf("查询错误: %v", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSubMenuNotFound
		}
		return err
	}

	return global.MySQLClient.Delete(&subMenu).Error
}

// UpdateMenuSort 更新菜单排序
func (m *menu) UpdateMenuSort(id uint, sort int) error {
	return global.MySQLClient.Model(&model.Menu{}).
		Where("id = ?", id).
		Update("sort", sort).Error
}

// UpdateSubMenuSort 更新子菜单排序
func (m *menu) UpdateSubMenuSort(id uint, sort int) error {
	return global.MySQLClient.Model(&model.SubMenu{}).
		Where("id = ?", id).
		Update("sort", sort).Error
}

// GetMenuList 获取菜单列表（表格中展示）
func (m *menu) GetMenuList(title string, page, limit int) (data *MenuList, err error) {

	// 定义数据的起始位置
	startSet := (page - 1) * limit

	// 定义返回的内容
	var (
		menus []*model.Menu
		total int64
	)

	// 获取菜单列表
	tx := global.MySQLClient.Model(&model.Menu{}).
		Preload("SubMenus", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort")
		}).                                   // 加载二级菜单，指定使用sort字段进行排序
		Where("title like ?", "%"+title+"%"). // 实现过滤
		Count(&total).                        // 获取一级菜单总数
		Limit(limit).
		Offset(startSet).
		Order("sort"). // 使用sort字段进行排序
		Find(&menus)
	if tx.Error != nil {
		return nil, err
	}

	return &MenuList{
		Items: menus,
		Total: total,
	}, nil
}

// GetUserMenu 获取用户有菜单（用户登录）
func (m *menu) GetUserMenu(tx *gorm.DB, username string) (data []*MenuItem, err error) {

	var (
		menus     []*model.Menu
		menuItems []*MenuItem
	)

	// 获取一级菜单
	if err := tx.Order("sort").Find(&menus).Error; err != nil {
		return nil, err
	}

	for _, menu := range menus {
		// 判断用户是否拥有该菜单权限
		ok, _ := global.CasBinServer.Enforce(username, menu.Name, "read")
		if ok {
			// 将一级菜单模型转换为返回给前端的格式
			menuItem := &MenuItem{
				Path:      menu.Path,
				Component: menu.Component,
				Name:      menu.Name,
				Redirect:  menu.Redirect,
				Meta: map[string]string{
					"title": menu.Title,
					"icon":  menu.Icon,
				},
				Children: nil,
			}

			// 获取一级菜单对应的二级菜单
			var subMenus []*model.SubMenu
			if err := tx.Where("menu_id = ?", menu.Id).Order("sort").Find(&subMenus).Error; err != nil {
				return nil, err
			}
			for _, subMenu := range subMenus {
				// 判断用户是否拥有该菜单权限
				ok, _ := global.CasBinServer.Enforce(username, subMenu.Name, "read")
				if ok {
					// 将二级菜单转换为返回给前端的格式
					subMenuItem := &MenuItem{
						Path:      subMenu.Path,
						Component: subMenu.Component,
						Name:      subMenu.Name,
						Redirect:  subMenu.Redirect,
						Meta: map[string]string{
							"title": subMenu.Title,
							"icon":  subMenu.Icon,
						},
					}

					// 将二级菜单添加到一级菜单的子菜单中
					menuItem.Children = append(menuItem.Children, subMenuItem)
				}
			}

			// 将一级菜单添加到返回给前端的菜单列表中
			menuItems = append(menuItems, menuItem)
		}
	}

	return menuItems, nil
}

// GetMenuTitle 根据菜单Name获取Title
func (m *menu) GetMenuTitle(menuName string) (title *string, err error) {
	if menuName == "" {
		return nil, fmt.Errorf("menuName 不能为空")
	}

	var menu model.Menu
	// 在一级菜单中查找
	err = global.MySQLClient.Where("name = ?", menuName).First(&menu).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("查询一级菜单失败: %w", err)
		}

		// 如果一级菜单没找到，查找二级菜单
		var subMenu model.SubMenu
		err = global.MySQLClient.Where("name = ?", menuName).First(&subMenu).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("在一级和二级菜单中都未找到名称为 %s 的记录", menuName)
			}
			return nil, fmt.Errorf("查询二级菜单失败: %w", err)
		}

		// 检查二级菜单的 Title 是否为空
		if subMenu.Title == "" {
			return nil, fmt.Errorf("二级菜单 %s 的标题为空", menuName)
		}
		return &subMenu.Title, nil
	}

	// 检查一级菜单的 Title 是否为空
	if menu.Title == "" {
		return nil, fmt.Errorf("一级菜单 %s 的标题为空", menuName)
	}
	return &menu.Title, nil
}
