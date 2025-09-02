# VGO Admin Gateway 模块化系统

## 概述

VGO Admin Gateway 采用了先进的模块化架构设计，支持插件式开发和动态加载。每个模块都可以独立开发、测试、打包和部署，为系统提供了极高的灵活性和可扩展性。

## 核心特性

### 🔧 模块化架构
- **独立开发**: 每个模块都是独立的Go项目，可以单独开发和维护
- **统一接口**: 所有模块都实现统一的`Module`接口，确保一致性
- **动态加载**: 支持运行时动态加载和卸载模块
- **依赖管理**: 支持模块间的依赖关系管理

### 📦 VKP打包系统
- **标准格式**: 使用`.vkp`（VGO Kernel Plugin）格式打包模块
- **元数据支持**: 包含完整的模块元数据信息
- **版本管理**: 支持语义化版本控制
- **依赖声明**: 明确声明模块依赖关系

### ⚙️ 配置管理
- **集中配置**: 统一的配置管理系统
- **动态更新**: 支持运行时配置更新
- **环境隔离**: 支持不同环境的配置隔离
- **健康检查**: 内置健康检查机制

### 🚀 独立运行模式
- **开发调试**: 模块可以独立运行，便于开发和调试
- **单元测试**: 支持模块级别的单元测试
- **集成测试**: 支持模块间的集成测试

## 系统架构

```
VGO Admin Gateway
├── Core System (核心系统)
│   ├── Module Manager (模块管理器)
│   ├── Plugin Loader (插件加载器)
│   ├── Config Manager (配置管理器)
│   └── Router (路由系统)
├── Modules (模块)
│   ├── IAM Module (身份认证模块)
│   ├── User Module (用户管理模块)
│   └── ... (其他模块)
└── VKP Packages (VKP包)
    ├── iam.vkp
    ├── user.vkp
    └── ...
```

## 快速开始

### 1. 创建新模块

```bash
# 创建模块目录
mkdir modules/your-module
cd modules/your-module

# 初始化Go模块
go mod init your-module

# 创建基本文件结构
mkdir -p cmd/server internal/handler internal/service
```

### 2. 实现模块接口

```go
package main

import (
    "github.com/vera-byte/vgo-gateway/internal/module"
    "github.com/gin-gonic/gin"
)

type YourModule struct {
    // 模块字段
}

func (m *YourModule) Name() string {
    return "your-module"
}

func (m *YourModule) Version() string {
    return "1.0.0"
}

func (m *YourModule) Description() string {
    return "Your module description"
}

func (m *YourModule) Initialize() error {
    // 初始化逻辑
    return nil
}

func (m *YourModule) RegisterRoutes(router *gin.RouterGroup) {
    // 注册路由
    router.GET("/your-endpoint", m.handleRequest)
}

func (m *YourModule) HealthCheck() error {
    // 健康检查逻辑
    return nil
}

func (m *YourModule) Shutdown() error {
    // 关闭逻辑
    return nil
}

// NewPlugin 插件入口函数
func NewPlugin() module.Module {
    return &YourModule{}
}
```

### 3. 创建构建脚本

```bash
#!/bin/bash
# build.sh

set -e

MODULE_NAME="your-module"
MODULE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$MODULE_DIR/build"
OUTPUT_DIR="$MODULE_DIR"

case "$1" in
    "build")
        echo "构建 $MODULE_NAME 模块..."
        mkdir -p "$BUILD_DIR"
        go build -buildmode=plugin -o "$BUILD_DIR/plugin" ./cmd/server
        ;;
    "package")
        echo "打包 $MODULE_NAME 模块..."
        ./build.sh build
        
        # 创建元数据文件
        cat > "$BUILD_DIR/metadata.json" << EOF
{
    "name": "$MODULE_NAME",
    "version": "1.0.0",
    "description": "Your module description",
    "author": "Your Name",
    "license": "MIT",
    "api_version": "v1",
    "gateway_min_version": "1.0.0",
    "standalone": true,
    "binary_name": "plugin"
}
EOF
        
        # 创建插件配置
        cat > "$BUILD_DIR/plugin.json" << EOF
{
    "name": "$MODULE_NAME",
    "version": "1.0.0",
    "description": "Your module description",
    "config_schema": {}
}
EOF
        
        # 打包为VKP文件
        cd "$BUILD_DIR"
        tar -czf "$OUTPUT_DIR/$MODULE_NAME.vkp" .
        echo "VKP包已创建: $OUTPUT_DIR/$MODULE_NAME.vkp"
        ;;
    "run")
        echo "独立运行 $MODULE_NAME 模块..."
        go run ./cmd/server/main.go standalone "$@"
        ;;
    *)
        echo "用法: $0 {build|package|run}"
        exit 1
        ;;
esac
```

### 4. 测试模块

```bash
# 独立运行测试
./build.sh run --port 8080

# 构建模块
./build.sh build

# 打包为VKP
./build.sh package
```

## 模块开发指南

### 目录结构

```
modules/your-module/
├── cmd/
│   └── server/
│       └── main.go          # 主程序入口
├── internal/
│   ├── handler/             # HTTP处理器
│   ├── service/             # 业务逻辑
│   ├── model/               # 数据模型
│   └── config/              # 配置定义
├── build.sh                 # 构建脚本
├── go.mod                   # Go模块定义
├── go.sum                   # 依赖校验
└── README.md                # 模块文档
```

### 最佳实践

1. **接口设计**
   - 保持接口简洁明了
   - 使用RESTful API设计原则
   - 提供完整的API文档

2. **错误处理**
   - 使用统一的错误处理机制
   - 提供有意义的错误信息
   - 记录详细的错误日志

3. **配置管理**
   - 使用环境变量进行配置
   - 提供合理的默认值
   - 支持配置验证

4. **日志记录**
   - 使用结构化日志
   - 记录关键操作和错误
   - 避免记录敏感信息

5. **测试覆盖**
   - 编写单元测试
   - 提供集成测试
   - 确保测试覆盖率

## 配置管理

### 模块配置文件

每个模块都有对应的配置文件 `configs/{module-name}.json`：

```json
{
  "name": "your-module",
  "version": "1.0.0",
  "enabled": true,
  "auto_start": true,
  "load_order": 100,
  "dependencies": ["iam"],
  "config": {
    "database_url": "postgres://localhost:5432/yourdb",
    "cache_ttl": 3600,
    "features": ["feature1", "feature2"]
  },
  "health_check": {
    "enabled": true,
    "interval": 30,
    "timeout": 10,
    "retries": 3
  }
}
```

### 配置字段说明

- `name`: 模块名称
- `version`: 模块版本
- `enabled`: 是否启用模块
- `auto_start`: 是否自动启动
- `load_order`: 加载顺序（数字越小越先加载）
- `dependencies`: 依赖的其他模块
- `config`: 模块特定配置
- `health_check`: 健康检查配置

## 部署指南

### 开发环境

```bash
# 启动网关
go run cmd/server/main.go

# 独立运行模块（用于开发调试）
cd modules/your-module
./build.sh run --port 8080
```

### 生产环境

```bash
# 构建所有模块
for module in modules/*/; do
    cd "$module"
    ./build.sh package
    cd -
done

# 启动网关（会自动加载VKP包）
./vgo-admin-gateway
```

## 故障排除

### 常见问题

1. **模块加载失败**
   - 检查VKP包格式是否正确
   - 验证模块依赖是否满足
   - 查看日志中的详细错误信息

2. **路由冲突**
   - 检查模块间是否有重复的路由定义
   - 使用不同的路由前缀

3. **配置错误**
   - 验证配置文件格式
   - 检查必需的配置项是否存在
   - 确认配置值的类型正确

### 调试技巧

1. **启用详细日志**
   ```bash
   export LOG_LEVEL=debug
   go run cmd/server/main.go
   ```

2. **独立运行模块**
   ```bash
   cd modules/your-module
   ./build.sh run --port 8080 --debug
   ```

3. **检查模块状态**
   ```bash
   curl http://localhost:8080/api/v1/modules
   ```

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License

## 联系我们

如有问题或建议，请提交 Issue 或联系开发团队。