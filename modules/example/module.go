package example

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ExampleModule 示例模块实现，展示如何实现module.BaseModule接口
type ExampleModule struct {
	config *Config
	logger *zap.Logger
}

// Config 示例模块配置
type Config struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	Message string `mapstructure:"message" json:"message"`
}

// NewExampleModule 创建新的示例模块
// 返回值: *ExampleModule 示例模块实例
func NewExampleModule() *ExampleModule {
	return &ExampleModule{}
}

// Name 获取模块名称
// 返回值: string 模块名称
func (m *ExampleModule) Name() string {
	return "example"
}

// Version 获取模块版本
// 返回值: string 模块版本
func (m *ExampleModule) Version() string {
	return "1.0.0"
}

// Description 获取模块描述
// 返回值: string 模块描述
func (m *ExampleModule) Description() string {
	return "Example module demonstrating how to implement BaseModule interface"
}

// Initialize 初始化模块
// 参数: ctx 上下文, config 模块配置, logger 日志器
// 返回值: error 错误信息
func (m *ExampleModule) Initialize(ctx context.Context, config interface{}, logger *zap.Logger) error {
	m.logger = logger

	// 解析配置
	configMap, ok := config.(map[string]interface{})
	if !ok {
		// 使用默认配置
		m.config = &Config{
			Enabled: true,
			Message: "Hello from Example Module!",
		}
	} else {
		m.config = &Config{
			Enabled: true,
			Message: "Hello from Example Module!",
		}

		if enabled, exists := configMap["enabled"]; exists {
			if enabledBool, ok := enabled.(bool); ok {
				m.config.Enabled = enabledBool
			}
		}

		if message, exists := configMap["message"]; exists {
			if messageStr, ok := message.(string); ok {
				m.config.Message = messageStr
			}
		}
	}

	m.logger.Info("Example module initialized", 
		zap.Bool("enabled", m.config.Enabled),
		zap.String("message", m.config.Message))

	return nil
}

// RegisterRoutes 注册模块路由
// 参数: router gin路由组, logger 日志器
// 返回值: error 错误信息
func (m *ExampleModule) RegisterRoutes(router *gin.RouterGroup, logger *zap.Logger) error {
	if !m.config.Enabled {
		logger.Info("Example module is disabled, skipping route registration")
		return nil
	}

	// 注册示例路由
	router.GET("/hello", m.helloHandler())
	router.GET("/info", m.infoHandler())

	logger.Info("Example module routes registered")
	return nil
}

// HealthCheck 健康检查
// 参数: ctx 上下文
// 返回值: error 错误信息
func (m *ExampleModule) HealthCheck(ctx context.Context) error {
	if !m.config.Enabled {
		return nil // 模块未启用时认为健康
	}
	return nil // 示例模块总是健康的
}

// Shutdown 关闭模块
// 参数: ctx 上下文
// 返回值: error 错误信息
func (m *ExampleModule) Shutdown(ctx context.Context) error {
	if m.logger != nil {
		m.logger.Info("Example module shutting down")
	}
	return nil
}

// helloHandler 处理hello请求
// 返回值: gin.HandlerFunc 处理函数
func (m *ExampleModule) helloHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": m.config.Message,
			"module":  m.Name(),
			"version": m.Version(),
		})
	}
}

// infoHandler 处理info请求
// 返回值: gin.HandlerFunc 处理函数
func (m *ExampleModule) infoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"name":        m.Name(),
				"version":     m.Version(),
				"description": m.Description(),
				"enabled":     m.config.Enabled,
				"config":      m.config,
			},
		})
	}
}