package module

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Manager 模块管理器
type Manager struct {
	modules map[string]Module
	logger  *zap.Logger
	mu      sync.RWMutex
}

// NewManager 创建新的模块管理器
// logger: 日志记录器
// 返回值: *Manager 管理器实例
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		modules: make(map[string]Module),
		logger:  logger,
	}
}

// RegisterModule 注册模块
// name: 模块名称
// module: 模块实例
// 返回值: error 错误信息
func (m *Manager) RegisterModule(name string, module Module) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.modules[name]; exists {
		return fmt.Errorf("module %s already registered", name)
	}

	m.modules[name] = module
	m.logger.Info("Module registered", zap.String("name", name))
	return nil
}

// UnregisterModule 注销模块
// name: 模块名称
// 返回值: error 错误信息
func (m *Manager) UnregisterModule(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.modules[name]; !exists {
		return fmt.Errorf("module %s not found", name)
	}

	delete(m.modules, name)
	m.logger.Info("Module unregistered", zap.String("name", name))
	return nil
}

// InitializeAll 初始化所有模块
// ctx: 上下文
// configs: 模块配置
// 返回值: error 错误信息
func (m *Manager) InitializeAll(ctx context.Context, configs map[string]interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, module := range m.modules {
		config := make(map[string]interface{})
		if moduleConfig, exists := configs[name]; exists {
			if configMap, ok := moduleConfig.(map[string]interface{}); ok {
				config = configMap
			}
		}

		if err := module.Initialize(ctx, config, m.logger); err != nil {
			return fmt.Errorf("failed to initialize module %s: %w", name, err)
		}
		m.logger.Info("Module initialized", zap.String("name", name))
	}

	return nil
}

// RegisterRoutes 注册所有模块的路由
// router: Gin路由组
// logger: 日志记录器
// 返回值: error 错误信息
func (m *Manager) RegisterRoutes(router *gin.RouterGroup, logger *zap.Logger) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, module := range m.modules {
		moduleGroup := router.Group("/" + name)
		if err := module.RegisterRoutes(moduleGroup, logger); err != nil {
			return fmt.Errorf("failed to register routes for module %s: %w", name, err)
		}
		logger.Info("Module routes registered", zap.String("name", name))
	}

	return nil
}

// GetModule 获取指定模块
// name: 模块名称
// 返回值: Module 模块实例, bool 是否存在
func (m *Manager) GetModule(name string) (Module, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	module, exists := m.modules[name]
	return module, exists
}

// ListModules 列出所有模块
// 返回值: []ModuleInfo 模块信息列表
func (m *Manager) ListModules() []ModuleInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var modules []ModuleInfo
	for name, module := range m.modules {
		modules = append(modules, ModuleInfo{
			Name:        name,
			Version:     module.Version(),
			Description: module.Description(),
		})
	}

	return modules
}

// HealthCheck 检查所有模块的健康状态
// ctx: 上下文
// 返回值: map[string]error 模块健康状态
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]error)
	for name, module := range m.modules {
		health[name] = module.HealthCheck(ctx)
	}

	return health
}

// ShutdownAll 关闭所有模块
// ctx: 上下文
// 返回值: error 错误信息
func (m *Manager) ShutdownAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errors []string
	for name, module := range m.modules {
		if err := module.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("module %s: %v", name, err))
			m.logger.Error("Failed to shutdown module", zap.String("name", name), zap.Error(err))
		} else {
			m.logger.Info("Module shutdown", zap.String("name", name))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %s", strings.Join(errors, "; "))
	}

	return nil
}