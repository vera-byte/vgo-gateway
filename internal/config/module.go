package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

// ModuleConfig 模块配置结构
type ModuleConfig struct {
	// Name 模块名称
	Name string `json:"name" yaml:"name"`
	
	// Version 模块版本
	Version string `json:"version" yaml:"version"`
	
	// Enabled 是否启用
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Config 模块特定配置
	Config map[string]interface{} `json:"config" yaml:"config"`
	
	// Dependencies 依赖的其他模块
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	
	// LoadOrder 加载顺序（数字越小越先加载）
	LoadOrder int `json:"load_order,omitempty" yaml:"load_order,omitempty"`
	
	// AutoStart 是否自动启动
	AutoStart bool `json:"auto_start" yaml:"auto_start"`
	
	// HealthCheck 健康检查配置
	HealthCheck *HealthCheckConfig `json:"health_check,omitempty" yaml:"health_check,omitempty"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	// Enabled 是否启用健康检查
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Interval 检查间隔（秒）
	Interval int `json:"interval" yaml:"interval"`
	
	// Timeout 超时时间（秒）
	Timeout int `json:"timeout" yaml:"timeout"`
	
	// Retries 重试次数
	Retries int `json:"retries" yaml:"retries"`
}

// ModuleConfigManager 模块配置管理器
type ModuleConfigManager struct {
	// configDir 配置文件目录
	configDir string
	
	// configs 已加载的配置映射
	configs map[string]*ModuleConfig
	
	// logger 日志记录器
	logger *zap.Logger
	
	// mu 读写锁
	mu sync.RWMutex
}

// NewModuleConfigManager 创建新的模块配置管理器
// configDir: 配置文件目录
// logger: 日志记录器
// 返回: 模块配置管理器实例
func NewModuleConfigManager(configDir string, logger *zap.Logger) *ModuleConfigManager {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	
	return &ModuleConfigManager{
		configDir: configDir,
		configs:   make(map[string]*ModuleConfig),
		logger:    logger,
	}
}

// LoadConfig 加载模块配置
// moduleName: 模块名称
// 返回: 模块配置和错误信息
func (m *ModuleConfigManager) LoadConfig(moduleName string) (*ModuleConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 检查是否已加载
	if config, exists := m.configs[moduleName]; exists {
		return config, nil
	}
	
	// 构建配置文件路径
	configPath := filepath.Join(m.configDir, moduleName+".json")
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果配置文件不存在，创建默认配置
		defaultConfig := m.createDefaultConfig(moduleName)
		m.configs[moduleName] = defaultConfig
		
		m.logger.Info("使用默认配置", 
			zap.String("module", moduleName),
			zap.String("config_path", configPath))
		
		return defaultConfig, nil
	}
	
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	
	// 解析配置
	var config ModuleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	
	// 验证配置
	if err := m.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	
	// 缓存配置
	m.configs[moduleName] = &config
	
	m.logger.Info("模块配置加载成功", 
		zap.String("module", moduleName),
		zap.String("version", config.Version),
		zap.Bool("enabled", config.Enabled))
	
	return &config, nil
}

// SaveConfig 保存模块配置
// config: 模块配置
// 返回: 错误信息
func (m *ModuleConfigManager) SaveConfig(config *ModuleConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 验证配置
	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	// 确保配置目录存在
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	
	// 构建配置文件路径
	configPath := filepath.Join(m.configDir, config.Name+".json")
	
	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	
	// 写入配置文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	
	// 更新缓存
	m.configs[config.Name] = config
	
	m.logger.Info("模块配置保存成功", 
		zap.String("module", config.Name),
		zap.String("config_path", configPath))
	
	return nil
}

// GetConfig 获取模块配置
// moduleName: 模块名称
// 返回: 模块配置和是否存在
func (m *ModuleConfigManager) GetConfig(moduleName string) (*ModuleConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	config, exists := m.configs[moduleName]
	return config, exists
}

// UpdateConfig 更新模块配置
// moduleName: 模块名称
// updates: 更新的配置项
// 返回: 错误信息
func (m *ModuleConfigManager) UpdateConfig(moduleName string, updates map[string]interface{}) error {
	m.mu.Lock()
	
	// 获取现有配置
	config, exists := m.configs[moduleName]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("模块配置不存在: %s", moduleName)
	}
	
	// 更新配置项
	for key, value := range updates {
		switch key {
		case "enabled":
			if enabled, ok := value.(bool); ok {
				config.Enabled = enabled
			}
		case "auto_start":
			if autoStart, ok := value.(bool); ok {
				config.AutoStart = autoStart
			}
		case "load_order":
			if loadOrder, ok := value.(int); ok {
				config.LoadOrder = loadOrder
			}
		case "config":
			if configMap, ok := value.(map[string]interface{}); ok {
				if config.Config == nil {
					config.Config = make(map[string]interface{})
				}
				for k, v := range configMap {
					config.Config[k] = v
				}
			}
		}
	}
	
	// 释放锁后保存配置
	m.mu.Unlock()
	return m.SaveConfig(config)
}

// DeleteConfig 删除模块配置
// moduleName: 模块名称
// 返回: 错误信息
func (m *ModuleConfigManager) DeleteConfig(moduleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 构建配置文件路径
	configPath := filepath.Join(m.configDir, moduleName+".json")
	
	// 删除配置文件
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除配置文件失败: %w", err)
	}
	
	// 从缓存中删除
	delete(m.configs, moduleName)
	
	m.logger.Info("模块配置删除成功", zap.String("module", moduleName))
	return nil
}

// ListConfigs 列出所有模块配置
// 返回: 模块配置列表
func (m *ModuleConfigManager) ListConfigs() []*ModuleConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	configs := make([]*ModuleConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, config)
	}
	
	return configs
}

// GetEnabledConfigs 获取已启用的模块配置
// 返回: 已启用的模块配置列表
func (m *ModuleConfigManager) GetEnabledConfigs() []*ModuleConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var enabledConfigs []*ModuleConfig
	for _, config := range m.configs {
		if config.Enabled {
			enabledConfigs = append(enabledConfigs, config)
		}
	}
	
	return enabledConfigs
}

// createDefaultConfig 创建默认配置
// moduleName: 模块名称
// 返回: 默认模块配置
func (m *ModuleConfigManager) createDefaultConfig(moduleName string) *ModuleConfig {
	return &ModuleConfig{
		Name:      moduleName,
		Version:   "1.0.0",
		Enabled:   true,
		AutoStart: true,
		LoadOrder: 100,
		Config:    make(map[string]interface{}),
		HealthCheck: &HealthCheckConfig{
			Enabled:  true,
			Interval: 30,
			Timeout:  10,
			Retries:  3,
		},
	}
}

// validateConfig 验证配置
// config: 模块配置
// 返回: 错误信息
func (m *ModuleConfigManager) validateConfig(config *ModuleConfig) error {
	if config.Name == "" {
		return fmt.Errorf("模块名称不能为空")
	}
	
	if config.Version == "" {
		return fmt.Errorf("模块版本不能为空")
	}
	
	if config.LoadOrder < 0 {
		return fmt.Errorf("加载顺序不能为负数")
	}
	
	if config.HealthCheck != nil {
		if config.HealthCheck.Interval <= 0 {
			return fmt.Errorf("健康检查间隔必须大于0")
		}
		if config.HealthCheck.Timeout <= 0 {
			return fmt.Errorf("健康检查超时时间必须大于0")
		}
		if config.HealthCheck.Retries < 0 {
			return fmt.Errorf("健康检查重试次数不能为负数")
		}
	}
	
	return nil
}

// ReloadConfig 重新加载模块配置
// moduleName: 模块名称
// 返回: 模块配置和错误信息
func (m *ModuleConfigManager) ReloadConfig(moduleName string) (*ModuleConfig, error) {
	m.mu.Lock()
	// 从缓存中删除
	delete(m.configs, moduleName)
	m.mu.Unlock()
	
	// 重新加载
	config, err := m.LoadConfig(moduleName)
	if err != nil {
		return nil, err
	}
	
	m.logger.Info("模块配置重新加载成功", zap.String("module", moduleName))
	return config, nil
}