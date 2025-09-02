package iam

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/vera-byte/vgo-gateway/pkg/client"
	"github.com/vera-byte/vgo-gateway/pkg/model"
	"github.com/vera-byte/vgo-gateway/internal/plugin"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// IAMModule IAM模块实现
// 实现了 module.BaseModule 和 plugin.Plugin 接口
type IAMModule struct {
	client      client.IAMClient
	config      *Config
	logger      *zap.Logger
	initialized bool
	metadata    *plugin.PluginMetadata
}

// Config IAM模块配置
type Config struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
	Timeout  int    `mapstructure:"timeout" json:"timeout"`
}

// NewIAMModule 创建新的IAM模块实例
// 返回: IAM模块实例
func NewIAMModule() *IAMModule {
	return &IAMModule{
		metadata: &plugin.PluginMetadata{
			Name:               "iam",
			Version:            "1.0.0",
			Description:        "Identity and Access Management module",
			Author:             "VGO Team",
			License:            "MIT",
			APIVersion:         "v1",
			MinGatewayVersion:  "1.0.0",
			Standalone:         true,
			Dependencies:       []string{},
		},
	}
}

// Name 获取模块名称
func (m *IAMModule) Name() string {
	return "iam"
}

// Version 获取模块版本
func (m *IAMModule) Version() string {
	return "1.0.0"
}

// Description 获取模块描述
func (m *IAMModule) Description() string {
	return "Identity and Access Management module for user authentication and authorization"
}

// InitializeModule 初始化模块（BaseModule接口）
func (m *IAMModule) InitializeModule(ctx context.Context, config interface{}, logger *zap.Logger) error {
	m.logger = logger

	// 解析配置
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config format for IAM module")
	}

	m.config = &Config{
		Endpoint: "localhost:9090", // 默认值
		Timeout:  30,               // 默认值
	}

	if endpoint, exists := configMap["endpoint"]; exists {
		if endpointStr, ok := endpoint.(string); ok {
			m.config.Endpoint = endpointStr
		}
	}

	if timeout, exists := configMap["timeout"]; exists {
		if timeoutInt, ok := timeout.(int); ok {
			m.config.Timeout = timeoutInt
		}
	}

	// 创建IAM客户端
	iamConfig := client.IAMConfig{
		Endpoint: m.config.Endpoint,
		Timeout:  m.config.Timeout,
	}

	client, err := client.NewIAMClient(iamConfig)
	if err != nil {
		return fmt.Errorf("failed to create IAM client: %w", err)
	}

	m.client = client
	
	// 只有logger不为nil时才记录日志
	if m.logger != nil {
		m.logger.Info("IAM module initialized", 
			zap.String("endpoint", m.config.Endpoint),
			zap.Int("timeout", m.config.Timeout))
	}

	m.initialized = true
	return nil
}

// GetName 获取插件名称（Plugin接口）
// 返回: 插件名称
func (m *IAMModule) GetName() string {
	return m.Name()
}

// GetVersion 获取插件版本（Plugin接口）
// 返回: 插件版本
func (m *IAMModule) GetVersion() string {
	return m.Version()
}

// GetDescription 获取插件描述（Plugin接口）
// 返回: 插件描述
func (m *IAMModule) GetDescription() string {
	return m.Description()
}

// GetMetadata 获取插件元数据
// 返回: 插件元数据
func (m *IAMModule) GetMetadata() *plugin.PluginMetadata {
	return m.metadata
}

// CanRunStandalone 是否支持独立运行
// 返回: 是否支持独立运行
func (m *IAMModule) CanRunStandalone() bool {
	return true
}

// Initialize 初始化插件（Plugin接口）
// ctx: 上下文
// logger: 日志器
// config: 配置数据
// 返回: 错误信息
func (m *IAMModule) Initialize(ctx context.Context, logger *zap.Logger, config interface{}) error {
	// 如果配置为nil，使用默认配置
	if config == nil {
		config = map[string]interface{}{
			"endpoint": "localhost:9090",
			"timeout":  30,
		}
	}
	return m.InitializeModule(ctx, config, logger)
}

// InitializeForModule 初始化模块（Module接口）
// ctx: 上下文
// config: 配置数据
// logger: 日志器
// 返回: 错误信息
func (m *IAMModule) InitializeForModule(ctx context.Context, config interface{}, logger *zap.Logger) error {
	// 如果配置为nil，使用默认配置
	if config == nil {
		config = map[string]interface{}{
			"endpoint": "localhost:9090",
			"timeout":  30,
		}
	}
	return m.InitializeModule(ctx, config, logger)
}

// RunStandalone 独立运行模式
// ctx: 上下文
// port: 监听端口
// 返回: 错误信息
func (m *IAMModule) RunStandalone(ctx context.Context, port int) error {
	// 如果logger为nil，创建一个默认的logger
	logger := m.logger
	if logger == nil {
		var err error
		logger, err = zap.NewDevelopment()
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}
	}
	
	// 使用插件独立运行器
	runner := plugin.NewStandaloneRunner(m, logger)
	return runner.Run(ctx, port)
}

// RegisterRoutes 注册路由（Plugin接口）
// router: Gin路由器
// 返回: 错误信息
func (m *IAMModule) RegisterRoutes(router *gin.Engine) error {
	// 创建API组
	api := router.Group("/api/v1/iam")
	return m.registerRoutesGroup(api, m.logger)
}

// Health 健康检查（Plugin接口）
// 返回: 健康状态和错误信息
// Health 健康检查（Plugin接口）
// 返回值: map[string]interface{} 健康状态信息, error 错误信息
func (m *IAMModule) Health() (map[string]interface{}, error) {
	if m.client == nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  "IAM client not initialized",
		}, fmt.Errorf("IAM client not initialized")
	}
	
	return map[string]interface{}{
		"status":      "healthy",
		"endpoint":    m.config.Endpoint,
		"initialized": m.initialized,
	}, nil
}



// registerRoutesGroup 注册模块路由（内部方法）
func (m *IAMModule) registerRoutesGroup(router *gin.RouterGroup, logger *zap.Logger) error {
	// 登录路由（公开）
	router.POST("/login", m.loginHandler())
	
	// 需要认证的路由
	auth := router.Group("")
	auth.Use(m.authMiddleware())
	{
		auth.GET("/profile", m.profileHandler())
		auth.POST("/logout", m.logoutHandler())
		auth.GET("/verify", m.verifyHandler())
	}

	// 管理员路由
	admin := auth.Group("/admin")
	admin.Use(m.requireRole("admin"))
	{
		admin.GET("/users", m.listUsersHandler())
		admin.POST("/users", m.createUserHandler())
		admin.PUT("/users/:id", m.updateUserHandler())
		admin.DELETE("/users/:id", m.deleteUserHandler())
	}

	// 只有logger不为nil时才记录日志
	if logger != nil {
		logger.Info("IAM module routes registered")
	}
	return nil
}

// HealthCheck 健康检查（BaseModule接口）
// 参数: ctx 上下文
// 返回值: error 错误信息
func (m *IAMModule) HealthCheck(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("IAM client not initialized")
	}
	if !m.initialized {
		return fmt.Errorf("IAM module not initialized")
	}
	return nil
}

// Shutdown 关闭模块
func (m *IAMModule) Shutdown(ctx context.Context) error {
	if m.client != nil {
		if closer, ok := m.client.(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}

// loginHandler 登录处理器
func (m *IAMModule) loginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			m.logger.Error("Invalid login request", zap.Error(err))
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Error:   err.Error(),
			})
			return
		}

		resp, err := m.client.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			m.logger.Error("Login failed", zap.Error(err), zap.String("username", req.Username))
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid credentials",
				Error:   err.Error(),
			})
			return
		}

		m.logger.Info("User logged in successfully", zap.String("username", req.Username))
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    http.StatusOK,
			Message: "Login successful",
			Data:    resp,
		})
	}
}

// profileHandler 获取用户资料处理器
func (m *IAMModule) profileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			})
			return
		}

		c.JSON(http.StatusOK, model.APIResponse{
			Code:    http.StatusOK,
			Message: "User profile retrieved successfully",
			Data:    user,
		})
	}
}

// logoutHandler 登出处理器
func (m *IAMModule) logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现登出逻辑（如令牌黑名单）
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    http.StatusOK,
			Message: "Logout successful",
		})
	}
}

// verifyHandler 验证令牌处理器
func (m *IAMModule) verifyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			})
			return
		}

		c.JSON(http.StatusOK, model.APIResponse{
			Code:    http.StatusOK,
			Message: "Token is valid",
			Data:    user,
		})
	}
}

// listUsersHandler 用户列表处理器
func (m *IAMModule) listUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现用户列表获取
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    http.StatusOK,
			Message: "Users retrieved successfully",
			Data:    []model.User{},
		})
	}
}

// createUserHandler 创建用户处理器
func (m *IAMModule) createUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现用户创建
		c.JSON(http.StatusCreated, model.APIResponse{
			Code:    http.StatusCreated,
			Message: "User created successfully",
		})
	}
}

// updateUserHandler 更新用户处理器
func (m *IAMModule) updateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现用户更新
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    http.StatusOK,
			Message: "User updated successfully",
		})
	}
}

// deleteUserHandler 删除用户处理器
func (m *IAMModule) deleteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现用户删除
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    http.StatusOK,
			Message: "User deleted successfully",
		})
	}
}

// authMiddleware 认证中间件
func (m *IAMModule) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Missing authorization header",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]
		user, err := m.client.VerifyToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid token",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

// requireRole 角色权限中间件
func (m *IAMModule) requireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		user, ok := userInterface.(*model.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Invalid user data",
			})
			c.Abort()
			return
		}

		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range user.Roles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, model.ErrorResponse{
				Code:    http.StatusForbidden,
				Message: "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}