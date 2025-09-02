# 模块开发指南

本文档介绍如何在VGO Admin Gateway中开发新的模块。

## 模块接口

所有模块都必须实现 `module.BaseModule` 接口：

```go
type BaseModule interface {
    // Name 获取模块名称
    Name() string
    
    // Version 获取模块版本
    Version() string
    
    // Description 获取模块描述
    Description() string
    
    // Initialize 初始化模块
    Initialize(ctx context.Context, config interface{}, logger *zap.Logger) error
    
    // RegisterRoutes 注册模块路由
    RegisterRoutes(router *gin.RouterGroup, logger *zap.Logger) error
    
    // Health 健康检查
    Health(ctx context.Context) error
    
    // Shutdown 关闭模块
    Shutdown(ctx context.Context) error
}
```

## 创建新模块

### 1. 创建模块目录

在 `modules/` 目录下创建新的模块目录，例如 `modules/mymodule/`。

### 2. 实现模块结构

创建 `module.go` 文件：

```go
package mymodule

import (
    "context"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// MyModule 我的模块实现
type MyModule struct {
    config *Config
    logger *zap.Logger
}

// Config 模块配置
type Config struct {
    Enabled bool   `mapstructure:"enabled" json:"enabled"`
    // 添加其他配置字段
}

// NewMyModule 创建新的模块实例
func NewMyModule() *MyModule {
    return &MyModule{}
}

// Name 获取模块名称
func (m *MyModule) Name() string {
    return "mymodule"
}

// Version 获取模块版本
func (m *MyModule) Version() string {
    return "1.0.0"
}

// Description 获取模块描述
func (m *MyModule) Description() string {
    return "My custom module description"
}

// Initialize 初始化模块
func (m *MyModule) Initialize(ctx context.Context, config interface{}, logger *zap.Logger) error {
    m.logger = logger
    
    // 解析配置
    configMap, ok := config.(map[string]interface{})
    if !ok {
        // 使用默认配置
        m.config = &Config{Enabled: true}
    } else {
        // 解析配置字段
        m.config = &Config{Enabled: true}
        if enabled, exists := configMap["enabled"]; exists {
            if enabledBool, ok := enabled.(bool); ok {
                m.config.Enabled = enabledBool
            }
        }
    }
    
    logger.Info("My module initialized")
    return nil
}

// RegisterRoutes 注册模块路由
func (m *MyModule) RegisterRoutes(router *gin.RouterGroup, logger *zap.Logger) error {
    if !m.config.Enabled {
        return nil
    }
    
    // 注册路由
    router.GET("/hello", m.helloHandler())
    
    logger.Info("My module routes registered")
    return nil
}

// Health 健康检查
func (m *MyModule) Health(ctx context.Context) error {
    // 实现健康检查逻辑
    return nil
}

// Shutdown 关闭模块
func (m *MyModule) Shutdown(ctx context.Context) error {
    if m.logger != nil {
        m.logger.Info("My module shutting down")
    }
    return nil
}

// helloHandler 示例处理函数
func (m *MyModule) helloHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello from my module!"})
    }
}
```

### 3. 创建模块工厂（可选）

创建 `factory.go` 文件：

```go
package mymodule

import (
    "github.com/vera-byte/vgo-gateway/internal/module"
)

// MyModuleFactory 模块工厂
type MyModuleFactory struct{}

// NewMyModuleFactory 创建新的模块工厂
func NewMyModuleFactory() *MyModuleFactory {
    return &MyModuleFactory{}
}

// CreateModule 创建模块实例
func (f *MyModuleFactory) CreateModule() (module.BaseModule, error) {
    return NewMyModule(), nil
}

// ModuleType 获取模块类型
func (f *MyModuleFactory) ModuleType() string {
    return "mymodule"
}
```

### 4. 注册模块

在 `cmd/server/main.go` 中注册模块：

```go
import (
    // ... 其他导入
    "github.com/vera-byte/vgo-gateway/modules/mymodule"
)

func main() {
    // ... 其他代码
    
    // 注册自定义模块
    logger.Info("Registering My module...")
    myModule := mymodule.NewMyModule()
    if err := moduleManager.RegisterModule("mymodule", myModule); err != nil {
        logger.Fatal("Failed to register My module", zap.Error(err))
    }
    
    // ... 其他代码
}
```

## 模块配置

在 `config/config.yaml` 中添加模块配置：

```yaml
modules:
  mymodule:
    enabled: true
    # 其他配置选项
```

## 最佳实践

### 1. 错误处理

- 在 `Initialize` 方法中进行充分的错误检查
- 使用结构化日志记录错误信息
- 返回有意义的错误消息

### 2. 配置管理

- 为所有配置项提供合理的默认值
- 使用 `mapstructure` 标签进行配置映射
- 验证配置的有效性

### 3. 路由注册

- 使用模块名作为路由前缀
- 实现适当的中间件
- 提供清晰的API文档

### 4. 健康检查

- 实现有意义的健康检查逻辑
- 检查外部依赖的连接状态
- 返回具体的错误信息

### 5. 优雅关闭

- 在 `Shutdown` 方法中清理资源
- 关闭数据库连接、文件句柄等
- 等待正在进行的操作完成

## 示例模块

项目中包含了一个完整的示例模块 (`modules/example/`)，展示了如何实现所有接口方法。你可以参考这个示例来创建自己的模块。

## 测试

为你的模块编写单元测试和集成测试：

```go
package mymodule

import (
    "context"
    "testing"
    "go.uber.org/zap"
)

func TestMyModule_Initialize(t *testing.T) {
    module := NewMyModule()
    logger, _ := zap.NewDevelopment()
    
    config := map[string]interface{}{
        "enabled": true,
    }
    
    err := module.Initialize(context.Background(), config, logger)
    if err != nil {
        t.Errorf("Initialize failed: %v", err)
    }
    
    if module.Name() != "mymodule" {
        t.Errorf("Expected name 'mymodule', got '%s'", module.Name())
    }
}
```

## 部署

1. 确保模块代码已提交到版本控制系统
2. 更新配置文件以启用新模块
3. 重新构建和部署应用程序
4. 验证模块是否正常工作