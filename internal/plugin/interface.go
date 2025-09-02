package plugin

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Plugin 插件接口，扩展了BaseModule接口
// 支持插件的生命周期管理和独立运行
type Plugin interface {
	// GetName 获取插件名称
	GetName() string
	
	// GetVersion 获取插件版本
	GetVersion() string
	
	// GetDescription 获取插件描述
	GetDescription() string
	
	// Initialize 初始化插件
	// ctx: 上下文
	// logger: 日志记录器
	// config: 配置数据
	// 返回: 错误信息
	Initialize(ctx context.Context, logger *zap.Logger, config interface{}) error
	
	// RegisterRoutes 注册路由
	// router: Gin路由器
	// 返回: 错误信息
	RegisterRoutes(router *gin.Engine) error
	
	// Health 健康检查
	// 返回: 健康状态和错误信息
	Health() (map[string]interface{}, error)
	
	// Shutdown 关闭插件
	// ctx: 上下文
	// 返回: 错误信息
	Shutdown(ctx context.Context) error
	
	// GetMetadata 获取插件元数据
	// 返回: 插件元数据
	GetMetadata() *PluginMetadata
	
	// CanRunStandalone 是否支持独立运行
	// 返回: 是否支持独立运行
	CanRunStandalone() bool
	
	// RunStandalone 独立运行模式
	// ctx: 上下文
	// port: 监听端口
	// 返回: 错误信息
	RunStandalone(ctx context.Context, port int) error
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	// Name 插件名称
	Name string `json:"name"`
	
	// Version 插件版本
	Version string `json:"version"`
	
	// Description 插件描述
	Description string `json:"description"`
	
	// Author 插件作者
	Author string `json:"author"`
	
	// License 许可证
	License string `json:"license"`
	
	// Dependencies 依赖项
	Dependencies []string `json:"dependencies"`
	
	// APIVersion API版本
	APIVersion string `json:"api_version"`
	
	// MinGatewayVersion 最小网关版本要求
	MinGatewayVersion string `json:"min_gateway_version"`
	
	// Standalone 是否支持独立运行
	Standalone bool `json:"standalone"`
	
	// ConfigSchema 配置模式
	ConfigSchema map[string]interface{} `json:"config_schema,omitempty"`
}

// PluginFactory 插件工厂接口
type PluginFactory interface {
	// CreatePlugin 创建插件实例
	// config: 插件配置
	// 返回: 插件实例和错误信息
	CreatePlugin(config interface{}) (Plugin, error)
	
	// GetPluginType 获取插件类型
	// 返回: 插件类型
	GetPluginType() string
	
	// GetMetadata 获取插件元数据
	// 返回: 插件元数据
	GetMetadata() *PluginMetadata
}

// PluginLoader 插件加载器接口
type PluginLoader interface {
	// LoadPlugin 加载插件
	// path: 插件文件路径
	// 返回: 插件实例和错误信息
	LoadPlugin(path string) (Plugin, error)
	
	// UnloadPlugin 卸载插件
	// name: 插件名称
	// 返回: 错误信息
	UnloadPlugin(name string) error
	
	// ListPlugins 列出已加载的插件
	// 返回: 插件名称列表
	ListPlugins() []string
	
	// GetPlugin 获取插件实例
	// name: 插件名称
	// 返回: 插件实例和是否存在
	GetPlugin(name string) (Plugin, bool)
}