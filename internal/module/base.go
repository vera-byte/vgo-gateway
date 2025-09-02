package module

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BaseModule 基础模块接口
// 所有模块都必须实现此接口
type BaseModule interface {
	// Name 获取模块名称
	// 返回值: string 模块名称
	Name() string
	
	// Version 获取模块版本
	// 返回值: string 模块版本
	Version() string
	
	// Description 获取模块描述
	// 返回值: string 模块描述
	Description() string
	
	// Initialize 初始化模块
	// 参数: ctx 上下文, config 模块配置, logger 日志器
	// 返回值: error 错误信息
	Initialize(ctx context.Context, config interface{}, logger *zap.Logger) error
	
	// RegisterRoutes 注册模块路由
	// 参数: router gin路由组, logger 日志器
	// 返回值: error 错误信息
	RegisterRoutes(router *gin.RouterGroup, logger *zap.Logger) error
	
	// HealthCheck 健康检查
	// 参数: ctx 上下文
	// 返回值: error 错误信息
	HealthCheck(ctx context.Context) error
	
	// Shutdown 关闭模块
	// 参数: ctx 上下文
	// 返回值: error 错误信息
	Shutdown(ctx context.Context) error
}

// ModuleFactory 模块工厂接口
// 用于创建模块实例
type ModuleFactory interface {
	// CreateModule 创建模块实例
	// 返回值: BaseModule 模块实例, error 错误信息
	CreateModule() (BaseModule, error)
	
	// ModuleType 获取模块类型
	// 返回值: string 模块类型
	ModuleType() string
}

// ModuleRegistry 模块注册表
// 用于注册和管理模块工厂
type ModuleRegistry struct {
	factories map[string]ModuleFactory
	logger    *zap.Logger
}

// NewModuleRegistry 创建新的模块注册表
// 参数: logger 日志器
// 返回值: *ModuleRegistry 注册表实例
func NewModuleRegistry(logger *zap.Logger) *ModuleRegistry {
	return &ModuleRegistry{
		factories: make(map[string]ModuleFactory),
		logger:    logger,
	}
}

// RegisterFactory 注册模块工厂
// 参数: moduleType 模块类型, factory 模块工厂
// 返回值: error 错误信息
func (r *ModuleRegistry) RegisterFactory(moduleType string, factory ModuleFactory) error {
	if _, exists := r.factories[moduleType]; exists {
		return fmt.Errorf("module factory %s already registered", moduleType)
	}
	
	r.factories[moduleType] = factory
	r.logger.Info("Module factory registered", zap.String("type", moduleType))
	return nil
}

// CreateModule 创建模块实例
// 参数: moduleType 模块类型
// 返回值: BaseModule 模块实例, error 错误信息
func (r *ModuleRegistry) CreateModule(moduleType string) (BaseModule, error) {
	factory, exists := r.factories[moduleType]
	if !exists {
		return nil, fmt.Errorf("module factory %s not found", moduleType)
	}
	
	return factory.CreateModule()
}

// ListFactories 列出所有注册的工厂
// 返回值: []string 工厂类型列表
func (r *ModuleRegistry) ListFactories() []string {
	types := make([]string, 0, len(r.factories))
	for moduleType := range r.factories {
		types = append(types, moduleType)
	}
	return types
}