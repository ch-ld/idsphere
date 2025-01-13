package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/axgle/mahonia"
	"github.com/xuri/excelize/v2"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"ops-api/dao"
	"ops-api/model"
	"ops-api/utils"
	ssh1 "ops-api/utils/ssh"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ListHostInput struct {
	HostGroupID uint   `form:"hostGroupId"`
	Keyword     string `form:"keyword"`
	Status      string `form:"status"`
	Region      string `form:"region"`
	Source      string `form:"source"`
	OSVersion   string `form:"osVersion"`
	Page        int    `form:"page" binding:"required,min=1"`
	PageSize    int    `form:"pageSize" binding:"required,min=1,max=100"`
}

// HostReq 主机创建请求信息
// @Description 主机创建请求信息
type HostReq struct {
	HostGroupID       uint              `json:"hostGroupId" binding:"required"`
	Hostname          string            `json:"hostname" binding:"required"`
	PrivateIP         string            `json:"privateIp" binding:"required,ip"`
	PublicIP          string            `json:"publicIp" binding:"omitempty,ip"`
	SSHPort           int               `json:"sshPort" binding:"required,min=1,max=65535"`
	SSHUser           string            `json:"sshUser" binding:"required"`
	SSHAuthType       model.SSHAuthType `json:"sshAuthType" binding:"required,oneof=password key"`
	SSHPassword       string            `json:"sshPassword"`
	SSHKey            string            `json:"sshKey"`
	IsPasswordChanged bool              `json:"isPasswordChanged"` // 添加一个标志位表示密码是否被修改
	OSType            string            `json:"osType" binding:"required"`
	OSVersion         string            `json:"osVersion"`
	Tags              []string          `json:"tags"`
	Description       string            `json:"description"`
	Region            string            `json:"region"`
	Status            model.HostStatus  `gorm:"type:varchar(20);default:'unknown'" json:"status"`
	Source            model.HostSource  `gorm:"type:varchar(20);not null" json:"source"`
}

// SSHConfig 包装 Sftp 配置
type SSHConfig struct {
	*ssh1.SSHClientConfig
}

// HostMetrics 主机指标结构体
type HostMetrics struct {
	CPU           int       `json:"cpu"`
	CPUUsage      float64   `json:"cpu_usage"`
	Memory        int       `json:"memory"`
	MemoryUsage   float64   `json:"memory_usage"`
	DiskSize      int       `json:"disk_size"`
	DiskUsage     float64   `json:"disk_usage"`
	OSType        string    `json:"os_type"`
	OSVersion     string    `json:"os_version"`
	KernelVersion string    `json:"kernel_version"`
	CollectedAt   time.Time `json:"collected_at"`
	Errors        []error   `json:"-"`
}

// HostRecord 表示导入的主机记录结构
type HostRecord struct {
	GroupName   string // 主机组名称
	Hostname    string // 主机名
	PrivateIP   string // 私有IP
	PublicIP    string // 公网IP
	SSHPort     string // SSH端口
	SSHUser     string // SSH用户
	AuthType    string // 认证类型
	Password    string // 密码
	SSHKey      string // SSH密钥
	OSType      string // 操作系统类型
	OSVersion   string // 操作系统版本
	Tags        string // 标签
	Description string // 描述
}

// OSInfo 操作系统信息结构体
type OSInfo struct {
	osType        string
	osVersion     string
	kernelVersion string
}

// AWS实例CPU和内存映射表（根据实例类型）
var awsInstanceCPUMemory = map[string]struct {
	CPU    int
	Memory int
}{
	// 通用型实例 - T系列（可突发性能实例）
	"t3.micro":   {CPU: 2, Memory: 1},
	"t3.small":   {CPU: 2, Memory: 2},
	"t3.medium":  {CPU: 2, Memory: 4},
	"t3.large":   {CPU: 2, Memory: 8},
	"t3.xlarge":  {CPU: 4, Memory: 16},
	"t3.2xlarge": {CPU: 8, Memory: 32},

	"t3a.micro":   {CPU: 2, Memory: 1},
	"t3a.small":   {CPU: 2, Memory: 2},
	"t3a.medium":  {CPU: 2, Memory: 4},
	"t3a.large":   {CPU: 2, Memory: 8},
	"t3a.xlarge":  {CPU: 4, Memory: 16},
	"t3a.2xlarge": {CPU: 8, Memory: 32},

	// 通用型实例 - M系列
	"m6i.large":    {CPU: 2, Memory: 8},
	"m6i.xlarge":   {CPU: 4, Memory: 16},
	"m6i.2xlarge":  {CPU: 8, Memory: 32},
	"m6i.4xlarge":  {CPU: 16, Memory: 64},
	"m6i.8xlarge":  {CPU: 32, Memory: 128},
	"m6i.12xlarge": {CPU: 48, Memory: 192},
	"m6i.16xlarge": {CPU: 64, Memory: 256},
	"m6i.24xlarge": {CPU: 96, Memory: 384},
	"m6i.32xlarge": {CPU: 128, Memory: 512},

	"m6a.large":    {CPU: 2, Memory: 8},
	"m6a.xlarge":   {CPU: 4, Memory: 16},
	"m6a.2xlarge":  {CPU: 8, Memory: 32},
	"m6a.4xlarge":  {CPU: 16, Memory: 64},
	"m6a.8xlarge":  {CPU: 32, Memory: 128},
	"m6a.12xlarge": {CPU: 48, Memory: 192},
	"m6a.16xlarge": {CPU: 64, Memory: 256},
	"m6a.24xlarge": {CPU: 96, Memory: 384},
	"m6a.32xlarge": {CPU: 128, Memory: 512},

	// 计算优化型实例 - C系列
	"c6i.large":    {CPU: 2, Memory: 4},
	"c6i.xlarge":   {CPU: 4, Memory: 8},
	"c6i.2xlarge":  {CPU: 8, Memory: 16},
	"c6i.4xlarge":  {CPU: 16, Memory: 32},
	"c6i.8xlarge":  {CPU: 32, Memory: 64},
	"c6i.12xlarge": {CPU: 48, Memory: 96},
	"c6i.16xlarge": {CPU: 64, Memory: 128},
	"c6i.24xlarge": {CPU: 96, Memory: 192},
	"c6i.32xlarge": {CPU: 128, Memory: 256},

	"c6a.large":    {CPU: 2, Memory: 4},
	"c6a.xlarge":   {CPU: 4, Memory: 8},
	"c6a.2xlarge":  {CPU: 8, Memory: 16},
	"c6a.4xlarge":  {CPU: 16, Memory: 32},
	"c6a.8xlarge":  {CPU: 32, Memory: 64},
	"c6a.12xlarge": {CPU: 48, Memory: 96},
	"c6a.16xlarge": {CPU: 64, Memory: 128},
	"c6a.24xlarge": {CPU: 96, Memory: 192},
	"c6a.32xlarge": {CPU: 128, Memory: 256},

	// 内存优化型实例 - R系列
	"r6i.large":    {CPU: 2, Memory: 16},
	"r6i.xlarge":   {CPU: 4, Memory: 32},
	"r6i.2xlarge":  {CPU: 8, Memory: 64},
	"r6i.4xlarge":  {CPU: 16, Memory: 128},
	"r6i.8xlarge":  {CPU: 32, Memory: 256},
	"r6i.12xlarge": {CPU: 48, Memory: 384},
	"r6i.16xlarge": {CPU: 64, Memory: 512},
	"r6i.24xlarge": {CPU: 96, Memory: 768},
	"r6i.32xlarge": {CPU: 128, Memory: 1024},

	// 存储优化型实例 - I系列
	"i4i.large":    {CPU: 2, Memory: 16},
	"i4i.xlarge":   {CPU: 4, Memory: 32},
	"i4i.2xlarge":  {CPU: 8, Memory: 64},
	"i4i.4xlarge":  {CPU: 16, Memory: 128},
	"i4i.8xlarge":  {CPU: 32, Memory: 256},
	"i4i.16xlarge": {CPU: 64, Memory: 512},
	"i4i.32xlarge": {CPU: 128, Memory: 1024},

	// GPU实例 - G系列
	"g5.xlarge":   {CPU: 4, Memory: 16},
	"g5.2xlarge":  {CPU: 8, Memory: 32},
	"g5.4xlarge":  {CPU: 16, Memory: 64},
	"g5.8xlarge":  {CPU: 32, Memory: 128},
	"g5.12xlarge": {CPU: 48, Memory: 192},
	"g5.16xlarge": {CPU: 64, Memory: 256},
	"g5.24xlarge": {CPU: 96, Memory: 384},
	"g5.48xlarge": {CPU: 192, Memory: 768},

	// GPU实例 - P系列
	"p4d.24xlarge":  {CPU: 96, Memory: 1152},
	"p4de.24xlarge": {CPU: 96, Memory: 1152},
	"p3.2xlarge":    {CPU: 8, Memory: 61},
	"p3.8xlarge":    {CPU: 32, Memory: 244},
	"p3.16xlarge":   {CPU: 64, Memory: 488},

	// 高内存实例 - X系列
	"x2idn.16xlarge": {CPU: 64, Memory: 1024},
	"x2idn.24xlarge": {CPU: 96, Memory: 1536},
	"x2idn.32xlarge": {CPU: 128, Memory: 2048},
	"x2iedn.xlarge":  {CPU: 4, Memory: 128},
	"x2iedn.2xlarge": {CPU: 8, Memory: 256},
	"x2iedn.4xlarge": {CPU: 16, Memory: 512},
	"x2iedn.8xlarge": {CPU: 32, Memory: 1024},
}

var Host host

type host struct{}

// HostCreate 创建主机
func (h *host) HostCreate(input *HostReq) error {
	if input == nil {
		return errors.New("主机信息不能为空")
	}
	// 验证并处理 Sftp 认证信息
	if err := h.validateAndEncryptSSHAuth(input); err != nil {
		return fmt.Errorf("SSH认证信息验证失败: %w", err)
	}
	// 设置主机默认属性
	h.setHostDefaultProperties(input)
	host := model.Host{
		HostGroupID: input.HostGroupID,
		Hostname:    input.Hostname,
		PrivateIP:   input.PrivateIP,
		PublicIP:    input.PublicIP,
		SSHPort:     input.SSHPort,
		SSHUser:     input.SSHUser,
		SSHAuthType: input.SSHAuthType,
		SSHPassword: input.SSHPassword,
		SSHKey:      input.SSHKey,
		OSType:      input.OSType,
		OSVersion:   input.OSVersion,
		Region:      input.Region,
		Description: input.Description,
		Tags:        strings.Join(input.Tags, ","),
		Source:      input.Source,
		Status:      input.Status,
	}
	return dao.Host.HostCreate(&host)
}

// HostUpdate 更新主机
func (h *host) HostUpdate(id uint, input *HostReq) error {
	if input == nil {
		return errors.New("更新信息不能为空")
	}
	// 验证并处理 Sftp 认证信息
	host := model.Host{
		ID:          id,
		HostGroupID: input.HostGroupID,
		Hostname:    input.Hostname,
		PrivateIP:   input.PrivateIP,
		PublicIP:    input.PublicIP,
		SSHPort:     input.SSHPort,
		SSHUser:     input.SSHUser,
		SSHAuthType: input.SSHAuthType,
		OSType:      input.OSType,
		OSVersion:   input.OSVersion,
		Region:      input.Region,
		SSHPassword: input.SSHKey,
		Description: input.Description,
		Tags:        strings.Join(input.Tags, ","),
	}

	if input.SSHPassword != "" {
		if input.IsPasswordChanged {
			if err := h.validateAndEncryptSSHAuth(input); err != nil {
				return fmt.Errorf("SSH认证信息验证失败: %w", err)
			}
		}
		host.SSHPassword = input.SSHPassword
	}
	return dao.Host.HostUpdate(id, &host)
}

// HostDelete 删除主机
func (h *host) HostDelete(id uint) error {
	return dao.Host.HostDelete(id)
}

// HostBatchDelete 批量删除主机
func (h *host) HostBatchDelete(ids []uint) error {
	return dao.Host.HostsBatchDelete(ids)
}

// HostsList 查询主机列表
func (h *host) HostsList(input *ListHostInput) ([]model.Host, int64, error) {
	params := map[string]interface{}{
		"hostGroupId": input.HostGroupID,
		"keyword":     input.Keyword,
		"status":      input.Status,
		"region":      input.Region,
		"source":      input.Source,
		"osVersion":   input.OSVersion,
	}
	offset := (input.Page - 1) * input.PageSize
	return dao.Host.HostsList(params, offset, input.PageSize)
}

// HostsImport 处理文件导入
func (h *host) HostsImport(reader io.Reader, filename string) error {
	// 根据文件扩展名选择不同的处理方式
	ext := strings.ToLower(filename)
	if strings.HasSuffix(ext, ".csv") {
		return processCSV(reader)
	} else if strings.HasSuffix(ext, ".xlsx") || strings.HasSuffix(ext, ".xls") {
		return processExcel(reader)
	}
	return fmt.Errorf("不支持的文件格式")
}

// processCSV 处理 CSV 文件
func processCSV(reader io.Reader) error {
	// 读取文件内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 移除 BOM 标记
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})

	// 使用 GBK 解码器
	decoder := mahonia.NewDecoder("gbk")
	if decoder == nil {
		return fmt.Errorf("创建GBK解码器失败")
	}

	// 转换为 UTF-8
	utf8Str := decoder.ConvertString(string(content))
	return processRecords(csv.NewReader(strings.NewReader(utf8Str)))
}

// processExcel 处理 Excel 文件
func processExcel(reader io.Reader) error {
	// 读取 Excel 文件
	xlsx, err := excelize.OpenReader(reader)
	if err != nil {
		return fmt.Errorf("打开Excel文件失败: %v", err)
	}
	defer xlsx.Close()

	// 获取第一个工作表
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("Excel文件没有工作表")
	}

	// 读取所有行
	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		return fmt.Errorf("读取Excel数据失败: %v", err)
	}

	return processExcelRows(rows)
}

// processRecords 处理记录（共用的处理逻辑）
func processRecords(reader *csv.Reader) error {
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	// 读取表头
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("读取表头失败: %v", err)
	}

	return processDataRows(headers, reader)
}

// processExcelRows 处理 Excel 行数据
func processExcelRows(rows [][]string) error {
	if len(rows) == 0 {
		return fmt.Errorf("Excel文件为空")
	}

	headers := rows[0]
	// 创建一个适配器，使Excel数据符合CSV reader的接口
	rowIndex := 1
	reader := &excelReader{
		rows:    rows,
		current: rowIndex,
	}

	return processDataRows(headers, reader)
}

// processDataRows 处理数据行（共用的业务逻辑）
func processDataRows(headers []string, reader interface{ Read() ([]string, error) }) error {
	// 验证表头
	expectedHeaders := []string{
		"主机组", "主机名", "私有IP", "公网IP", "SSH端口",
		"SSH用户名", "认证类型", "密码", "密钥", "操作系统", "系统版本",
		"标签", "描述",
	}
	if len(headers) != len(expectedHeaders) {
		fmt.Println("表格格式不正确，请使用正确的导入模板")
	}

	var hosts []*model.Host
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("第 %d 行数据解析失败: %v", lineNum, err)
		}

		// 验证必填字段
		if len(record) < 3 || record[0] == "" || record[1] == "" || record[2] == "" {
			return fmt.Errorf("第 %d 行数据不完整，主机组、主机名和私有IP为必填项", lineNum)
		}

		// 处理主机组
		host, err := processHostRecord(record, lineNum)
		if err != nil {
			return err
		}
		hosts = append(hosts, host)
		lineNum++
	}

	if len(hosts) == 0 {
		return fmt.Errorf("文件中没有有效的主机数据")
	}

	// 批量创建主机
	return dao.Host.HostsBatchCreate(hosts)
}

// processHostRecord 处理单条主机记录
func processHostRecord(record []string, lineNum int) (*model.Host, error) {
	// 1. 验证记录长度
	if len(record) < 13 { // 增加了SSH密钥字段
		return nil, fmt.Errorf("第 %d 行数据字段不足，期望13个字段", lineNum)
	}

	// 2. 解析记录到结构体
	hostRecord := &HostRecord{
		GroupName:   strings.TrimSpace(record[0]),
		Hostname:    strings.TrimSpace(record[1]),
		PrivateIP:   strings.TrimSpace(record[2]),
		PublicIP:    strings.TrimSpace(record[3]),
		SSHPort:     strings.TrimSpace(record[4]),
		SSHUser:     strings.TrimSpace(record[5]),
		AuthType:    strings.TrimSpace(record[6]),
		Password:    strings.TrimSpace(record[7]),
		SSHKey:      strings.TrimSpace(record[8]),
		OSType:      strings.TrimSpace(record[9]),
		OSVersion:   strings.TrimSpace(record[10]),
		Tags:        strings.TrimSpace(record[11]),
		Description: strings.TrimSpace(record[12]),
	}

	// 3. 验证必填字段
	if err := validateRequiredFields(hostRecord, lineNum); err != nil {
		return nil, err
	}

	// 4. 处理主机组
	groupID, err := processHostGroup(hostRecord.GroupName)
	if err != nil {
		return nil, fmt.Errorf("第 %d 行处理主机组失败: %w", lineNum, err)
	}

	// 5. 处理SSH配置
	sshConfig, err := processSSHConfig(hostRecord, lineNum)
	if err != nil {
		return nil, fmt.Errorf("第 %d 行处理SSH配置失败: %w", lineNum, err)
	}

	// 6. 创建主机记录
	host := &model.Host{
		HostGroupID: groupID,
		Hostname:    hostRecord.Hostname,
		PrivateIP:   hostRecord.PrivateIP,
		PublicIP:    hostRecord.PublicIP,
		SSHPort:     sshConfig.Port,
		SSHUser:     sshConfig.UserName,
		SSHAuthType: model.SSHAuthType(sshConfig.AuthModel),
		SSHPassword: sshConfig.Password,
		SSHKey:      sshConfig.PublicKey,
		OSType:      hostRecord.OSType,
		OSVersion:   hostRecord.OSVersion,
		Tags:        hostRecord.Tags,
		Description: hostRecord.Description,
		Source:      model.HostSourceImport,
		Status:      model.HostStatusUnknown,
	}

	// 7. 验证主机记录的完整性
	if err := dao.Host.ValidateHost(host, lineNum); err != nil {
		return nil, err
	}

	return host, nil
}

// processSSHConfig 处理SSH配置
func processSSHConfig(record *HostRecord, lineNum int) (*ssh1.SSHClientConfig, error) {
	config := &ssh1.SSHClientConfig{
		Port:     22, // 默认端口
		UserName: record.SSHUser,
	}

	// 处理SSH端口
	if record.SSHPort != "" {
		port, err := strconv.Atoi(record.SSHPort)
		if err != nil {
			return nil, fmt.Errorf("SSH端口格式错误: %w", err)
		}
		if port <= 0 || port > 65535 {
			return nil, fmt.Errorf("SSH端口范围无效（1-65535）")
		}
		config.Port = port
	}

	// 验证认证类型
	if record.AuthType == "" {
		return nil, fmt.Errorf("认证类型不能为空")
	}
	if !isValidAuthType(record.AuthType) {
		return nil, fmt.Errorf("无效的认证类型: %s", record.AuthType)
	}
	config.AuthModel = record.AuthType

	// 根据认证类型处理凭证
	switch strings.ToUpper(config.AuthModel) {
	case "PASSWORD":
		if record.Password == "" {
			return nil, fmt.Errorf("密码认证方式下密码不能为空")
		}
		encryptedPass, err := utils.Encrypt(record.Password)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %w", err)
		}
		config.Password = encryptedPass

	case "PUBLICKEY":
		if record.SSHKey == "" {
			return nil, fmt.Errorf("密钥认证方式下SSH密钥不能为空")
		}
		// 验证SSH密钥格式
		if err := validateSSHKey(record.SSHKey); err != nil {
			return nil, fmt.Errorf("无效的SSH密钥: %w", err)
		}
		encryptedKey, err := utils.Encrypt(record.SSHKey)
		if err != nil {
			return nil, fmt.Errorf("SSH密钥加密失败: %w", err)
		}
		config.PublicKey = encryptedKey

	default:
		return nil, fmt.Errorf("不支持的认证类型: %s", record.AuthType)
	}

	return config, nil
}

// processHostGroup 处理主机组
func processHostGroup(groupName string) (uint, error) {
	groupID, err := dao.HostGroup.GetByName(groupName)
	if err == nil {
		return groupID, nil
	}

	hostGroup := &model.HostGroup{
		Name: groupName,
	}
	if err := dao.HostGroup.Create(hostGroup); err != nil {
		return 0, fmt.Errorf("创建主机组失败: %w", err)
	}
	return hostGroup.ID, nil
}

// validateSSHKey 验证SSH密钥格式
func validateSSHKey(key string) error {
	// 验证是否是有效的SSH私钥格式
	if !strings.HasPrefix(key, "-----BEGIN") || !strings.HasSuffix(key, "-----END") {
		return errors.New("无效的SSH密钥格式")
	}
	return nil
}

// isValidAuthType 验证认证类型是否有效
func isValidAuthType(authType string) bool {
	validTypes := []string{"password", "key"} // 根据实际情况修改
	for _, t := range validTypes {
		if authType == t {
			return true
		}
	}
	return false
}

// validateRequiredFields 验证必填字段
func validateRequiredFields(record *HostRecord, lineNum int) error {
	if record.GroupName == "" {
		return fmt.Errorf("第 %d 行主机组名称不能为空", lineNum)
	}
	if record.Hostname == "" {
		return fmt.Errorf("第 %d 行主机名不能为空", lineNum)
	}
	if record.PrivateIP == "" {
		return fmt.Errorf("第 %d 行私有IP不能为空", lineNum)
	}
	return nil
}

// excelReader 实现类似CSV reader的接口
type excelReader struct {
	rows    [][]string
	current int
}

func (e *excelReader) Read() ([]string, error) {
	if e.current >= len(e.rows) {
		return nil, io.EOF
	}
	row := e.rows[e.current]
	e.current++
	return row, nil
}

// CollectHostInfo 收集主机信息
func (h *host) CollectHostInfo(hostID uint) error {
	// 1. 获取主机信息
	host, err := dao.Host.HostGetByID(hostID)
	if err != nil {
		return fmt.Errorf("获取主机信息失败: %w", err)
	}
	if host == nil {
		return errors.New("主机不存在")
	}

	// 2. 创建 Sftp 配置
	config, err := createSSHConfig(host)
	if err != nil {
		return fmt.Errorf("创建SSH配置失败: %w", err)
	}

	// 3. 并发收集信息
	info, err := collectHostMetrics(config)
	if err != nil {
		return fmt.Errorf("收集主机指标失败: %w", err)
	}

	// 4. 更新数据库
	return updateHostInfo(hostID, info)
}

// createSSHConfig 根据主机认证类型创建 Sftp 配置
func createSSHConfig(host *model.Host) (*SSHConfig, error) {
	config := &ssh1.SSHClientConfig{
		Timeout:  time.Second * 5,
		IP:       host.PrivateIP,
		Port:     host.SSHPort,
		UserName: host.SSHUser,
	}

	switch host.SSHAuthType {
	case model.SSHAuthPassword:
		password, err := utils.Decrypt(host.SSHPassword)
		if err != nil {
			return nil, fmt.Errorf("解密SSH密码失败: %w", err)
		}
		config.Password = password
		config.AuthModel = "PASSWORD"
	case model.SSHAuthKey:
		sshKey, err := utils.Decrypt(host.SSHKey)
		if err != nil {
			return nil, fmt.Errorf("解密SSH密钥失败: %w", err)
		}
		config.PublicKey = sshKey
		config.AuthModel = "PUBLICKEY"
	default:
		return nil, fmt.Errorf("不支持的SSH认证类型: %v", host.SSHAuthType)
	}
	return &SSHConfig{config}, nil
}

// collectHostMetrics 并发收集主机指标
func collectHostMetrics(config *SSHConfig) (*HostMetrics, error) {
	metrics := &HostMetrics{
		CollectedAt: time.Now(),
	}

	// 使用 errgroup 进行并发控制
	g := new(errgroup.Group)
	var mu sync.Mutex // 保护 metrics 的并发访问

	// 收集 CPU 信息
	g.Go(func() error {
		cpu, cpuUsage, err := collectCPUInfo(config)
		if err != nil {
			return fmt.Errorf("收集CPU信息失败: %w", err)
		}
		mu.Lock()
		metrics.CPU = cpu
		metrics.CPUUsage = cpuUsage
		mu.Unlock()
		return nil
	})

	// 收集内存信息
	g.Go(func() error {
		memory, memoryUsage, err := collectMemoryInfo(config)
		if err != nil {
			return fmt.Errorf("收集内存信息失败: %w", err)
		}
		mu.Lock()
		metrics.Memory = memory
		metrics.MemoryUsage = memoryUsage
		mu.Unlock()
		return nil
	})

	// 收集磁盘信息
	g.Go(func() error {
		diskSize, diskUsage, err := collectDiskInfo(config)
		if err != nil {
			return fmt.Errorf("收集磁盘信息失败: %w", err)
		}
		mu.Lock()
		metrics.DiskSize = diskSize
		metrics.DiskUsage = diskUsage
		mu.Unlock()
		return nil
	})

	// 收集操作系统信息
	g.Go(func() error {
		osInfo, err := collectOSInfo(config)
		if err != nil {
			return fmt.Errorf("收集操作系统信息失败: %w", err)
		}
		mu.Lock()
		metrics.OSType = osInfo.osType
		metrics.OSVersion = osInfo.osVersion
		metrics.KernelVersion = osInfo.kernelVersion
		mu.Unlock()
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return metrics, nil
}

// 各个收集函数的实现
func collectCPUInfo(config *SSHConfig) (cpu int, usage float64, err error) {
	cmd := `nproc && top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\([0-9.]*\)%* id.*/\1/' | awk '{print 100 -$1}'`
	output, err := ssh1.SshCommand(config.SSHClientConfig, cmd)
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("无效的CPU信息输出")
	}

	cpu, err = strconv.Atoi(lines[0])
	if err != nil {
		return 0, 0, fmt.Errorf("解析CPU核数失败: %w", err)
	}

	usage, err = strconv.ParseFloat(lines[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("解析CPU使用率失败: %w", err)
	}

	return cpu, usage, nil
}

// collectMemoryInfo 收集内存信息
func collectMemoryInfo(config *SSHConfig) (memory int, usage float64, err error) {
	// 使用 free 命令获取内存信息
	// -m 以MB为单位显示
	// NR==2 选择第二行（实际内存使用行）
	// $2 总内存, $3 已用内存
	cmd := `free -m | awk 'NR==2{printf "%d %.2f", $2, $3/$2 * 100}'`

	output, err := ssh1.SshCommand(config.SSHClientConfig, cmd)
	if err != nil {
		return 0, 0, fmt.Errorf("执行内存信息命令失败: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return 0, 0, errors.New("获取内存信息为空")
	}

	var memTotal int
	var memUsage float64
	_, err = fmt.Sscanf(output, "%d %f", &memTotal, &memUsage)
	if err != nil {
		return 0, 0, fmt.Errorf("解析内存信息失败: %w", err)
	}

	// 验证数据合理性
	if memTotal <= 0 || memUsage < 0 || memUsage > 100 {
		return 0, 0, fmt.Errorf("无效的内存数据: total=%d, usage=%.2f", memTotal, memUsage)
	}

	return memTotal, memUsage, nil
}

// collectDiskInfo 收集磁盘信息
func collectDiskInfo(config *SSHConfig) (size int, usage float64, err error) {
	// df 命令获取根分区信息
	// $2 总大小(KB), $3 已用空间, $5 使用百分比
	cmd := `df / | awk 'NR==2{printf "%d %.2f", $2/1024/1024, $5}'`

	output, err := ssh1.SshCommand(config.SSHClientConfig, cmd)
	if err != nil {
		return 0, 0, fmt.Errorf("执行磁盘信息命令失败: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return 0, 0, errors.New("获取磁盘信息为空")
	}

	var diskSize int
	var diskUsage float64
	_, err = fmt.Sscanf(output, "%d %f", &diskSize, &diskUsage)
	if err != nil {
		return 0, 0, fmt.Errorf("解析磁盘信息失败: %w", err)
	}

	// 验证数据合理性
	if diskSize <= 0 || diskUsage < 0 || diskUsage > 100 {
		return 0, 0, fmt.Errorf("无效的磁盘数据: size=%d, usage=%.2f", diskSize, diskUsage)
	}

	return diskSize, diskUsage, nil
}

// collectOSInfo 收集操作系统信息
func collectOSInfo(config *SSHConfig) (*OSInfo, error) {
	// 构建一个多命令组合
	// 1. uname -o: 获取操作系统类型
	// 2. lsb_release -d: 获取发行版信息
	// 3. uname -r: 获取内核版本
	cmd := strings.Join([]string{
		`uname -o | awk -F '/' '{print $NF}'`,                                     // OS类型
		`(lsb_release -d 2>/dev/null || cat /etc/*release | head -n 1) | cut -f2`, // OS版本
		`uname -r`, // 内核版本
	}, " && echo '---' && ")

	output, err := ssh1.SshCommand(config.SSHClientConfig, cmd)
	if err != nil {
		return nil, fmt.Errorf("执行系统信息命令失败: %w", err)
	}

	// 分割输出
	parts := strings.Split(strings.TrimSpace(output), "---")
	if len(parts) != 3 {
		return nil, fmt.Errorf("解析系统信息失败，unexpected output format: %s", output)
	}

	info := &OSInfo{
		osType:        strings.TrimSpace(parts[0]),
		osVersion:     strings.TrimSpace(parts[1]),
		kernelVersion: strings.TrimSpace(parts[2]),
	}

	// 验证数据完整性
	if info.osType == "" || info.osVersion == "" || info.kernelVersion == "" {
		return nil, errors.New("系统信息不完整")
	}

	// 清理和规范化版本信息
	info.osVersion = cleanOSVersion(info.osVersion)
	info.kernelVersion = cleanKernelVersion(info.kernelVersion)

	return info, nil
}

// cleanOSVersion 清理和规范化操作系统版本信息
func cleanOSVersion(version string) string {
	// 移除常见的前缀
	prefixes := []string{
		"Description:",
		"PRETTY_NAME=",
		"NAME=",
		"VERSION=",
	}

	version = strings.TrimSpace(version)
	for _, prefix := range prefixes {
		version = strings.TrimPrefix(version, prefix)
	}

	// 移除引号
	version = strings.Trim(version, `"'`)

	return strings.TrimSpace(version)
}

// cleanKernelVersion 清理和规范化内核版本信息
func cleanKernelVersion(version string) string {
	// 只保留主要版本信息，移除额外的构建信息
	if idx := strings.Index(version, "-"); idx != -1 {
		version = version[:idx]
	}
	return strings.TrimSpace(version)
}

// 更新主机信息到数据库
func updateHostInfo(hostID uint, metrics *HostMetrics) error {
	updates := &model.Host{
		ID:            hostID,
		Status:        model.HostStatusOnline,
		LastCheckTime: &metrics.CollectedAt,
		CPU:           metrics.CPU,
		CPUUsage:      metrics.CPUUsage,
		Memory:        metrics.Memory,
		MemoryUsage:   metrics.MemoryUsage,
		DiskSize:      metrics.DiskSize,
		DiskUsage:     metrics.DiskUsage,
		OSType:        metrics.OSType,
		OSVersion:     metrics.OSVersion,
		KernelVersion: metrics.KernelVersion,
	}

	if err := dao.Host.HostUpdate(hostID, updates); err != nil {
		return fmt.Errorf("更新主机信息失败: %w", err)
	}

	return nil
}

// validateAndEncryptSSHAuth 验证并加密 Sftp 认证信息
func (h *host) validateAndEncryptSSHAuth(host *HostReq) error {
	switch host.SSHAuthType {
	case model.SSHAuthPassword:
		return h.handlePasswordAuth(host)
	case model.SSHAuthKey:
		return h.handleKeyAuth(host)
	case "": // 如果认证类型为空
		return errors.New("SSH认证类型不能为空")
	default:
		return fmt.Errorf("不支持的SSH认证类型: %s", host.SSHAuthType)
	}
}

// handlePasswordAuth 处理密码认证方式
func (h *host) handlePasswordAuth(host *HostReq) error {
	if host.SSHPassword == "" {
		return errors.New("SSH密码不能为空")
	}

	// 加密密码
	encryptedPassword, err := utils.Encrypt(host.SSHPassword)
	if err != nil {
		return fmt.Errorf("加密SSH密码失败: %w", err)
	}

	host.SSHPassword = encryptedPassword
	// 确保密钥字段为空
	host.SSHKey = ""
	return nil
}

// handleKeyAuth 处理密钥认证方式
func (h *host) handleKeyAuth(host *HostReq) error {
	if host.SSHKey == "" {
		return errors.New("SSH密钥不能为空")
	}
	// 加密密钥
	encryptedKey, err := utils.Encrypt(host.SSHKey)
	if err != nil {
		return fmt.Errorf("加密SSH密钥失败: %w", err)
	}

	host.SSHKey = encryptedKey
	// 确保密码字段为空
	host.SSHPassword = ""
	return nil
}

// setHostDefaultProperties 设置主机默认属性
func (h *host) setHostDefaultProperties(host *HostReq) {
	host.Source = model.HostSourceManual
	host.Status = model.HostStatusUnknown
	// 可以添加其他默认属性设置
	// 如果需要设置其他默认值，可以在这里添加
}

// 同步云主机
func (h *host) SyncCloudHosts(provider string, config map[string]string, hostGroupId int) error {
	regions := strings.Split(config["regions"], ",")
	var wg sync.WaitGroup
	errChan := make(chan error, len(regions))
	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			var err error
			switch provider {
			case "aliyun":
				err = syncAliyunHosts(config["accessKey"], config["accessSecret"], region, hostGroupId)
			case "aws":
				err = syncAWSHosts(config["accessKey"], config["accessSecret"], region, hostGroupId)
			}
			if err != nil {
				errChan <- fmt.Errorf("region %s sync failed: %v", region, err)
			}
		}(region)
	}
	// 等待所有同步完成
	wg.Wait()
	close(errChan)
	// 收集错误
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}
	if len(errors) > 0 {
		return fmt.Errorf("同步出现错误: %s", strings.Join(errors, "; "))
	}
	return nil
}

// 同步阿里云主机
func syncAliyunHosts(accessKey, accessSecret, region string, hostGroupId int) error {
	if accessKey == "" || accessSecret == "" || region == "" {
		return fmt.Errorf("参数不能为空: accessKey, accessSecret, region")
	}

	client, err := ecs.NewClientWithAccessKey(region, accessKey, accessSecret)
	if err != nil {
		return fmt.Errorf("创建阿里云客户端失败: %v", err)
	}

	request := ecs.CreateDescribeInstancesRequest()
	request.RegionId = region
	// 设置较大的页面大小以获取所有实例
	request.PageSize = requests.NewInteger(100)

	response, err := client.DescribeInstances(request)
	if err != nil {
		return fmt.Errorf("获取实例列表失败: %v", err)
	}

	if response == nil || len(response.Instances.Instance) == 0 {
		return fmt.Errorf("未找到任何实例")
	}

	for _, instance := range response.Instances.Instance {
		// 获取私网IP
		var privateIP string
		if len(instance.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
			privateIP = instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
		}

		// 获取公网IP (优先获取弹性公网IP)
		var publicIP string
		if instance.EipAddress.IpAddress != "" {
			publicIP = instance.EipAddress.IpAddress
		} else if len(instance.PublicIpAddress.IpAddress) > 0 {
			publicIP = instance.PublicIpAddress.IpAddress[0]
		}

		// 检查必要字段
		if instance.InstanceName == "" {
			instance.InstanceName = instance.InstanceId // 如果实例名为空，使用实例ID
		}

		// 处理操作系统信息
		osType := instance.OSType
		if osType == "" {
			osType = "Unknown"
		}
		osVersion := instance.OSName
		if osVersion == "" {
			osVersion = "Unknown"
		}
		t := time.Now()
		host := &model.Host{
			Hostname:        instance.InstanceName,
			HostGroupID:     uint(hostGroupId),
			PrivateIP:       privateIP,
			PublicIP:        publicIP,
			CPU:             instance.Cpu,
			Memory:          instance.Memory,
			OSType:          osType,
			OSVersion:       osVersion,
			Status:          convertAliyunStatus(instance.Status),
			Source:          model.HostSourceAliyun,
			CloudInstanceID: &instance.InstanceId,
			Region:          region,
			Tags:            convertAliyunTags(instance.Tags.Tag),
			LastCheckTime:   &t,
		}

		// 检查必要字段是否为空
		if err := validateHost(host); err != nil {
			log.Printf("警告: 实例 %s 数据验证失败: %v", instance.InstanceId, err)
			continue // 跳过此实例，继续处理下一个
		}

		// 尝试创建或更新主机记录
		HostExists, HostId := dao.Host.IsHostnameExists(host.Hostname)
		if HostExists {
			host.ID = HostId
			err := dao.Host.HostUpdate(HostId, host)
			if err != nil {
				log.Printf("警告: 更新实例 %s 失败: %v", instance.InstanceId, err)
				continue
			}
		} else {
			err := dao.Host.HostCreate(host)
			if err != nil {
				log.Printf("警告: 保存实例 %s 失败: %v", instance.InstanceId, err)
				continue
			}
		}
	}
	return nil
}

// 同步AWS主机
func syncAWSHosts(accessKey, accessSecret, region string, hostGroupId int) error {
	if accessKey == "" || accessSecret == "" || region == "" {
		return fmt.Errorf("参数不能为空: accessKey, accessSecret, region")
	}

	// 创建AWS配置
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			accessKey, accessSecret, "",
		))),
	)
	if err != nil {
		return fmt.Errorf("配置AWS客户端失败: %v", err)
	}

	// 创建EC2客户端
	client := ec2.NewFromConfig(cfg)
	input := &ec2.DescribeInstancesInput{}

	// 获取实例列表
	result, err := client.DescribeInstances(context.Background(), input)
	if err != nil {
		return fmt.Errorf("获取AWS实例列表失败: %v", err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			// 检查必要字段
			if instance.InstanceId == nil {
				continue
			}

			// 获取私网IP
			privateIP := ""
			if instance.PrivateIpAddress != nil {
				privateIP = *instance.PrivateIpAddress
			}

			// 获取公网IP
			publicIP := ""
			if instance.PublicIpAddress != nil {
				publicIP = *instance.PublicIpAddress
			}

			// 获取主机名
			hostname := getAWSTagValue(instance.Tags, "Name")
			if hostname == "" {
				hostname = *instance.InstanceId
			}
			t := time.Now()
			host := &model.Host{
				Hostname:    hostname,
				HostGroupID: uint(hostGroupId),
				PrivateIP:   privateIP,
				PublicIP:    publicIP,
				CPU:         awsInstanceCPUMemory[string(instance.InstanceType)].CPU,
				Memory:      awsInstanceCPUMemory[string(instance.InstanceType)].Memory * 1024,
				//OSType:          determineAWSOSType(instance),
				OSVersion:       aws.ToString(instance.PlatformDetails),
				Status:          convertAWSStatus(instance.State.Name),
				Source:          model.HostSourceAWS,
				CloudInstanceID: instance.InstanceId,
				Region:          region,
				Tags:            convertAWSTags(instance.Tags),
				LastCheckTime:   &t,
			}

			// 验证主机信息
			if err := validateHost(host); err != nil {
				continue
			}
			// 尝试创建或更新主机记录
			HostExists, HostId := dao.Host.IsHostnameExists(host.Hostname)
			if HostExists {
				host.ID = HostId
				err := dao.Host.HostUpdate(HostId, host)
				if err != nil {
					log.Printf("警告: 更新实例 %s 失败: %v", instance.InstanceId, err)
					continue
				}
			} else {
				err := dao.Host.HostCreate(host)
				if err != nil {
					log.Printf("警告: 保存实例 %s 失败: %v", instance.InstanceId, err)
					continue
				}
			}
		}
	}

	return nil
}

// 获取AWS标签值
func getAWSTagValue(tags []types.Tag, key string) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == key {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

// 转换AWS状态到系统状态
func convertAWSStatus(state types.InstanceStateName) model.HostStatus {
	switch state {
	case types.InstanceStateNameRunning:
		return model.HostStatusOnline
	case types.InstanceStateNameStopped:
		return model.HostStatusOffline
	case types.InstanceStateNamePending:
		return model.HostStatusOnline
	case types.InstanceStateNameStopping:
		return model.HostStatusOffline
	default:
		return model.HostStatusUnknown
	}
}

// 转换AWS标签到JSON字符串
func convertAWSTags(tags []types.Tag) string {
	tagsMap := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagsMap[*tag.Key] = *tag.Value
		}
	}

	jsonBytes, err := json.Marshal(tagsMap)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

// 辅助函数
func convertAliyunStatus(status string) model.HostStatus {
	switch status {
	case "Running":
		return model.HostStatusOnline
	case "Stopped":
		return model.HostStatusOffline
	default:
		return model.HostStatusUnknown
	}
}

func convertAliyunTags(tags []ecs.Tag) string {
	var tagPairs []string
	for _, tag := range tags {
		tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", tag.TagKey, tag.TagValue))
	}
	return strings.Join(tagPairs, ",")
}

// 验证主机信息
func validateHost(host *model.Host) error {
	if host.CloudInstanceID == nil || *host.CloudInstanceID == "" {
		return fmt.Errorf("云实例ID不能为空")
	}
	if host.Hostname == "" {
		return fmt.Errorf("主机名不能为空")
	}
	if host.PrivateIP == "" {
		return fmt.Errorf("私网IP不能为空")
	}
	if host.CPU <= 0 {
		return fmt.Errorf("CPU核数必须大于0")
	}
	if host.Memory <= 0 {
		return fmt.Errorf("内存大小必须大于0")
	}
	return nil
}
