package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Manager 插件管理器
// 负责插件的注册、加载、卸载和生命周期管理
type Manager struct {
	// plugins 已注册的插件映射
	plugins map[string]Plugin
	
	// factories 插件工厂映射
	factories map[string]PluginFactory
	
	// loader 插件加载器
	loader PluginLoader
	
	// installer 插件安装器
	installer *PluginInstaller
	
	// logger 日志记录器
	logger *zap.Logger
	
	// mu 读写锁
	mu sync.RWMutex
	
	// initialized 是否已初始化
	initialized bool
}

// NewManager 创建新的插件管理器
// logger: 日志记录器
// vpksDir: vpks目录路径
// 返回: 插件管理器实例
func NewManager(logger *zap.Logger, vpksDir string) *Manager {
	return &Manager{
		plugins:   make(map[string]Plugin),
		factories: make(map[string]PluginFactory),
		installer: NewPluginInstaller(vpksDir, logger),
		logger:    logger,
	}
}

// SetLoader 设置插件加载器
// loader: 插件加载器
func (m *Manager) SetLoader(loader PluginLoader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loader = loader
}

// RegisterFactory 注册插件工厂
// factory: 插件工厂
// 返回: 错误信息
func (m *Manager) RegisterFactory(factory PluginFactory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	pluginType := factory.GetPluginType()
	if _, exists := m.factories[pluginType]; exists {
		return fmt.Errorf("plugin factory for type '%s' already registered", pluginType)
	}
	
	m.factories[pluginType] = factory
	m.logger.Info("Plugin factory registered", zap.String("type", pluginType))
	return nil
}

// RegisterPlugin 注册插件实例
// plugin: 插件实例
// 返回: 错误信息
func (m *Manager) RegisterPlugin(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := plugin.GetName()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin '%s' already registered", name)
	}
	
	m.plugins[name] = plugin
	m.logger.Info("Plugin registered", 
		zap.String("name", name),
		zap.String("version", plugin.GetVersion()))
	return nil
}

// LoadPluginFromFile 从文件加载插件
// path: 插件文件路径
// 返回: 错误信息
func (m *Manager) LoadPluginFromFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.loader == nil {
		return fmt.Errorf("plugin loader not set")
	}
	
	plugin, err := m.loader.LoadPlugin(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin from %s: %w", path, err)
	}
	
	name := plugin.GetName()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin '%s' already loaded", name)
	}
	
	m.plugins[name] = plugin
	m.logger.Info("Plugin loaded from file", 
		zap.String("name", name),
		zap.String("path", path))
	return nil
}

// UnregisterPlugin 注销插件
// name: 插件名称
// 返回: 错误信息
func (m *Manager) UnregisterPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin '%s' not found", name)
	}
	
	// 关闭插件
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := plugin.Shutdown(ctx); err != nil {
		m.logger.Warn("Failed to shutdown plugin", 
			zap.String("name", name),
			zap.Error(err))
	}
	
	delete(m.plugins, name)
	m.logger.Info("Plugin unregistered", zap.String("name", name))
	return nil
}

// InitializeAll 初始化所有插件
// ctx: 上下文
// config: 配置映射
// 返回: 错误信息
func (m *Manager) InitializeAll(ctx context.Context, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for name, plugin := range m.plugins {
		pluginConfig := config[name]
		if err := plugin.Initialize(ctx, m.logger, pluginConfig); err != nil {
			return fmt.Errorf("failed to initialize plugin '%s': %w", name, err)
		}
		m.logger.Info("Plugin initialized", zap.String("name", name))
	}
	
	m.initialized = true
	return nil
}

// RegisterAllRoutes 注册所有插件的路由
// router: Gin路由器
// 返回: 错误信息
func (m *Manager) RegisterAllRoutes(router *gin.Engine) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for name, plugin := range m.plugins {
		if err := plugin.RegisterRoutes(router); err != nil {
			return fmt.Errorf("failed to register routes for plugin '%s': %w", name, err)
		}
		m.logger.Info("Plugin routes registered", zap.String("name", name))
	}
	
	return nil
}

// HealthCheckAll 检查所有插件的健康状态
// 返回: 健康状态映射和错误信息
func (m *Manager) HealthCheckAll() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]interface{})
	allHealthy := true
	
	for name, plugin := range m.plugins {
		health, err := plugin.Health()
		if err != nil {
			result[name] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			allHealthy = false
		} else {
			result[name] = health
		}
	}
	
	result["overall_status"] = "healthy"
	if !allHealthy {
		result["overall_status"] = "unhealthy"
	}
	
	return result, nil
}

// ShutdownAll 关闭所有插件
// ctx: 上下文
// 返回: 错误信息
func (m *Manager) ShutdownAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var errors []error
	
	for name, plugin := range m.plugins {
		if err := plugin.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown plugin '%s': %w", name, err))
		} else {
			m.logger.Info("Plugin shutdown", zap.String("name", name))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}
	
	m.initialized = false
	return nil
}

// GetPlugin 获取插件实例
// name: 插件名称
// 返回: 插件实例和是否存在
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	plugin, exists := m.plugins[name]
	return plugin, exists
}

// ListPlugins 列出所有已注册的插件
// 返回: 插件信息列表
func (m *Manager) ListPlugins() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var plugins []map[string]interface{}
	for name, plugin := range m.plugins {
		metadata := plugin.GetMetadata()
		plugins = append(plugins, map[string]interface{}{
			"name":        name,
			"version":     plugin.GetVersion(),
			"description": plugin.GetDescription(),
			"metadata":    metadata,
		})
	}
	
	return plugins
}

// IsInitialized 检查是否已初始化
// 返回: 是否已初始化
func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

// InstallPluginFromURL 从URL安装插件
// ctx: 上下文
// pluginURL: 插件下载URL
// 返回: 错误信息
func (m *Manager) InstallPluginFromURL(ctx context.Context, pluginURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 下载插件
	localPath, err := m.installer.InstallFromURL(ctx, pluginURL)
	if err != nil {
		return fmt.Errorf("安装插件失败: %w", err)
	}
	
	m.logger.Info("插件下载完成", 
		zap.String("url", pluginURL),
		zap.String("local_path", localPath))
	
	return nil
}

// InstallAndLoadPluginFromURL 从URL安装并加载插件
// ctx: 上下文
// pluginURL: 插件下载URL
// 返回: 错误信息
func (m *Manager) InstallAndLoadPluginFromURL(ctx context.Context, pluginURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 下载插件
	localPath, err := m.installer.InstallFromURL(ctx, pluginURL)
	if err != nil {
		return fmt.Errorf("安装插件失败: %w", err)
	}
	
	// 加载插件
	plugin, err := m.loader.LoadPlugin(localPath)
	if err != nil {
		return fmt.Errorf("加载插件失败: %w", err)
	}
	
	name := plugin.GetName()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("插件 '%s' 已存在", name)
	}
	
	m.plugins[name] = plugin
	m.logger.Info("插件安装并加载成功", 
		zap.String("name", name),
		zap.String("url", pluginURL),
		zap.String("local_path", localPath))
	
	return nil
}

// ListInstalledPlugins 列出已安装的插件文件
// 返回: 插件文件列表和错误信息
func (m *Manager) ListInstalledPlugins() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.installer.ListInstalledPlugins()
}

// RemoveInstalledPlugin 移除已安装的插件文件
// filename: 插件文件名
// 返回: 错误信息
func (m *Manager) RemoveInstalledPlugin(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.installer.RemovePlugin(filename)
}