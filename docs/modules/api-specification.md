# VGO Admin Gateway 模块化系统 API 规范

## 概述

本文档定义了VGO Admin Gateway模块化系统的API规范，包括核心系统API和模块开发API标准。

## 核心系统API

### 1. 模块管理API

#### 1.1 获取模块列表

```http
GET /api/v1/modules
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "name": "iam",
      "version": "1.0.0",
      "description": "身份认证和授权模块",
      "status": "running",
      "health": "healthy",
      "routes": [
        "/api/v1/auth/login",
        "/api/v1/auth/logout",
        "/api/v1/auth/refresh"
      ],
      "dependencies": [],
      "load_order": 10,
      "auto_start": true,
      "standalone_support": true
    },
    {
      "name": "user",
      "version": "1.0.0",
      "description": "用户管理模块",
      "status": "running",
      "health": "healthy",
      "routes": [
        "/api/v1/users",
        "/api/v1/users/{id}",
        "/api/v1/users/search"
      ],
      "dependencies": ["iam"],
      "load_order": 20,
      "auto_start": true,
      "standalone_support": true
    }
  ]
}
```

#### 1.2 获取模块详情

```http
GET /api/v1/modules/{module_name}
```

**路径参数：**
- `module_name`: 模块名称

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "name": "iam",
    "version": "1.0.0",
    "description": "身份认证和授权模块",
    "author": "VGO Team",
    "license": "MIT",
    "api_version": "v1",
    "gateway_min_version": "1.0.0",
    "status": "running",
    "health": "healthy",
    "uptime": "2h30m15s",
    "memory_usage": "45.2MB",
    "cpu_usage": "2.1%",
    "request_count": 1250,
    "error_count": 3,
    "last_error": null,
    "config": {
      "jwt_secret": "***",
      "token_expire": 3600,
      "refresh_expire": 86400
    },
    "routes": [
      {
        "method": "POST",
        "path": "/api/v1/auth/login",
        "description": "用户登录"
      },
      {
        "method": "POST",
        "path": "/api/v1/auth/logout",
        "description": "用户登出"
      },
      {
        "method": "POST",
        "path": "/api/v1/auth/refresh",
        "description": "刷新令牌"
      }
    ]
  }
}
```

#### 1.3 加载模块

```http
POST /api/v1/modules/{module_name}/load
```

**请求体：**
```json
{
  "vkp_path": "/path/to/module.vkp",
  "config": {
    "enabled": true,
    "auto_start": true,
    "load_order": 100
  }
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "模块加载成功",
  "data": {
    "name": "new-module",
    "status": "loaded",
    "load_time": "2023-12-01T10:30:00Z"
  }
}
```

#### 1.4 卸载模块

```http
DELETE /api/v1/modules/{module_name}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "模块卸载成功",
  "data": {
    "name": "module-name",
    "status": "unloaded",
    "unload_time": "2023-12-01T10:35:00Z"
  }
}
```

#### 1.5 启动模块

```http
POST /api/v1/modules/{module_name}/start
```

**响应示例：**
```json
{
  "code": 200,
  "message": "模块启动成功",
  "data": {
    "name": "module-name",
    "status": "running",
    "start_time": "2023-12-01T10:40:00Z"
  }
}
```

#### 1.6 停止模块

```http
POST /api/v1/modules/{module_name}/stop
```

**响应示例：**
```json
{
  "code": 200,
  "message": "模块停止成功",
  "data": {
    "name": "module-name",
    "status": "stopped",
    "stop_time": "2023-12-01T10:45:00Z"
  }
}
```

#### 1.7 重启模块

```http
POST /api/v1/modules/{module_name}/restart
```

**响应示例：**
```json
{
  "code": 200,
  "message": "模块重启成功",
  "data": {
    "name": "module-name",
    "status": "running",
    "restart_time": "2023-12-01T10:50:00Z"
  }
}
```

### 2. 配置管理API

#### 2.1 获取模块配置

```http
GET /api/v1/modules/{module_name}/config
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "name": "iam",
    "version": "1.0.0",
    "enabled": true,
    "auto_start": true,
    "load_order": 10,
    "dependencies": [],
    "config": {
      "jwt_secret": "your-secret-key",
      "token_expire": 3600,
      "refresh_expire": 86400,
      "database_url": "postgres://localhost:5432/iam"
    },
    "health_check": {
      "enabled": true,
      "interval": 30,
      "timeout": 10,
      "retries": 3
    }
  }
}
```

#### 2.2 更新模块配置

```http
PUT /api/v1/modules/{module_name}/config
```

**请求体：**
```json
{
  "enabled": true,
  "auto_start": false,
  "load_order": 15,
  "config": {
    "token_expire": 7200,
    "new_setting": "value"
  }
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "配置更新成功",
  "data": {
    "name": "iam",
    "updated_at": "2023-12-01T11:00:00Z",
    "restart_required": true
  }
}
```

### 3. 健康检查API

#### 3.1 系统健康检查

```http
GET /api/v1/health
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": "healthy",
    "timestamp": "2023-12-01T11:05:00Z",
    "uptime": "5h30m45s",
    "version": "1.0.0",
    "modules": {
      "total": 3,
      "running": 3,
      "stopped": 0,
      "error": 0
    },
    "system": {
      "memory_usage": "256MB",
      "cpu_usage": "15.2%",
      "goroutines": 45,
      "gc_cycles": 123
    }
  }
}
```

#### 3.2 模块健康检查

```http
GET /api/v1/modules/{module_name}/health
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "name": "iam",
    "status": "healthy",
    "timestamp": "2023-12-01T11:05:00Z",
    "uptime": "5h30m45s",
    "checks": [
      {
        "name": "database",
        "status": "healthy",
        "response_time": "2ms",
        "last_check": "2023-12-01T11:05:00Z"
      },
      {
        "name": "redis",
        "status": "healthy",
        "response_time": "1ms",
        "last_check": "2023-12-01T11:05:00Z"
      }
    ],
    "metrics": {
      "request_count": 1250,
      "error_count": 3,
      "avg_response_time": "45ms",
      "memory_usage": "45.2MB"
    }
  }
}
```

### 4. 路由管理API

#### 4.1 获取所有路由

```http
GET /api/v1/routes
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "method": "POST",
      "path": "/api/v1/auth/login",
      "module": "iam",
      "handler": "LoginHandler",
      "middleware": ["cors", "rate_limit"],
      "description": "用户登录",
      "request_count": 450,
      "avg_response_time": "120ms"
    },
    {
      "method": "GET",
      "path": "/api/v1/users",
      "module": "user",
      "handler": "ListUsersHandler",
      "middleware": ["cors", "auth", "rate_limit"],
      "description": "获取用户列表",
      "request_count": 230,
      "avg_response_time": "85ms"
    }
  ]
}
```

#### 4.2 获取模块路由

```http
GET /api/v1/modules/{module_name}/routes
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "method": "POST",
      "path": "/api/v1/auth/login",
      "handler": "LoginHandler",
      "middleware": ["cors", "rate_limit"],
      "description": "用户登录"
    },
    {
      "method": "POST",
      "path": "/api/v1/auth/logout",
      "handler": "LogoutHandler",
      "middleware": ["cors", "auth"],
      "description": "用户登出"
    }
  ]
}
```

## 模块开发API标准

### 1. 模块接口规范

每个模块必须实现以下接口：

```go
type Module interface {
    // Name 返回模块名称
    Name() string
    
    // Version 返回模块版本
    Version() string
    
    // Description 返回模块描述
    Description() string
    
    // Initialize 初始化模块
    Initialize() error
    
    // RegisterRoutes 注册路由
    RegisterRoutes(router *gin.RouterGroup)
    
    // HealthCheck 健康检查
    HealthCheck() error
    
    // Shutdown 关闭模块
    Shutdown() error
}
```

### 2. 标准HTTP响应格式

所有模块的API响应都应遵循统一格式：

#### 2.1 成功响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    // 响应数据
  },
  "timestamp": "2023-12-01T11:10:00Z",
  "request_id": "req-123456789"
}
```

#### 2.2 分页响应

```json
{
  "code": 200,
  "message": "success",
  "data": [
    // 数据列表
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total": 100,
    "total_pages": 10
  },
  "timestamp": "2023-12-01T11:10:00Z",
  "request_id": "req-123456789"
}
```

#### 2.3 错误响应

```json
{
  "code": 400,
  "message": "请求参数错误",
  "error": {
    "type": "validation_error",
    "details": [
      {
        "field": "email",
        "message": "邮箱格式不正确"
      }
    ]
  },
  "timestamp": "2023-12-01T11:10:00Z",
  "request_id": "req-123456789"
}
```

### 3. HTTP状态码规范

| 状态码 | 含义 | 使用场景 |
|--------|------|----------|
| 200 | OK | 请求成功 |
| 201 | Created | 资源创建成功 |
| 204 | No Content | 请求成功，无返回内容 |
| 400 | Bad Request | 请求参数错误 |
| 401 | Unauthorized | 未认证 |
| 403 | Forbidden | 无权限 |
| 404 | Not Found | 资源不存在 |
| 409 | Conflict | 资源冲突 |
| 422 | Unprocessable Entity | 请求格式正确但语义错误 |
| 429 | Too Many Requests | 请求过于频繁 |
| 500 | Internal Server Error | 服务器内部错误 |
| 502 | Bad Gateway | 网关错误 |
| 503 | Service Unavailable | 服务不可用 |

### 4. 请求/响应头规范

#### 4.1 标准请求头

```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <token>
X-Request-ID: req-123456789
X-Client-Version: 1.0.0
User-Agent: VGO-Client/1.0.0
```

#### 4.2 标准响应头

```http
Content-Type: application/json; charset=utf-8
X-Request-ID: req-123456789
X-Response-Time: 45ms
X-Rate-Limit-Remaining: 99
X-Rate-Limit-Reset: 1701423600
Cache-Control: no-cache
```

### 5. 认证和授权

#### 5.1 JWT Token格式

```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user123",
    "iat": 1701420000,
    "exp": 1701423600,
    "iss": "vgo-admin-gateway",
    "aud": "vgo-modules",
    "roles": ["admin", "user"],
    "permissions": ["read:users", "write:users"]
  }
}
```

#### 5.2 权限检查中间件

模块应使用统一的权限检查中间件：

```go
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 权限检查逻辑
    }
}

// 使用示例
router.GET("/users", RequirePermission("read:users"), handler.ListUsers)
```

### 6. 错误处理规范

#### 6.1 错误类型定义

```go
type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Type    string `json:"type"`
    Details interface{} `json:"details,omitempty"`
}

// 预定义错误类型
var (
    ErrValidation    = &APIError{Code: 400, Type: "validation_error"}
    ErrUnauthorized  = &APIError{Code: 401, Type: "unauthorized"}
    ErrForbidden     = &APIError{Code: 403, Type: "forbidden"}
    ErrNotFound      = &APIError{Code: 404, Type: "not_found"}
    ErrConflict      = &APIError{Code: 409, Type: "conflict"}
    ErrInternal      = &APIError{Code: 500, Type: "internal_error"}
)
```

#### 6.2 统一错误处理中间件

```go
func ErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            // 错误处理逻辑
        }
    }
}
```

### 7. 日志规范

#### 7.1 日志格式

使用结构化日志（JSON格式）：

```json
{
  "timestamp": "2023-12-01T11:15:00Z",
  "level": "info",
  "module": "user",
  "message": "用户创建成功",
  "request_id": "req-123456789",
  "user_id": "user123",
  "duration": "45ms",
  "method": "POST",
  "path": "/api/v1/users",
  "status": 201
}
```

#### 7.2 日志级别

- `DEBUG`: 调试信息
- `INFO`: 一般信息
- `WARN`: 警告信息
- `ERROR`: 错误信息
- `FATAL`: 致命错误

### 8. 性能监控

#### 8.1 指标收集

每个模块应收集以下指标：

- 请求数量
- 响应时间
- 错误率
- 内存使用
- CPU使用
- 数据库连接数

#### 8.2 监控端点

```http
GET /metrics
```

返回Prometheus格式的指标数据。

### 9. 配置规范

#### 9.1 配置文件格式

使用JSON格式的配置文件：

```json
{
  "name": "module-name",
  "version": "1.0.0",
  "enabled": true,
  "auto_start": true,
  "load_order": 100,
  "dependencies": ["iam"],
  "config": {
    "database_url": "${DATABASE_URL}",
    "redis_url": "${REDIS_URL}",
    "log_level": "info"
  },
  "health_check": {
    "enabled": true,
    "interval": 30,
    "timeout": 10,
    "retries": 3
  }
}
```

#### 9.2 环境变量支持

配置值支持环境变量替换，格式：`${VAR_NAME}`

### 10. 测试规范

#### 10.1 单元测试

每个模块应包含完整的单元测试：

```go
func TestUserService_CreateUser(t *testing.T) {
    // 测试逻辑
}
```

#### 10.2 集成测试

提供集成测试脚本：

```bash
#!/bin/bash
# integration-test.sh

# 启动测试环境
docker-compose -f docker-compose.test.yml up -d

# 运行测试
go test ./test/integration/... -v

# 清理测试环境
docker-compose -f docker-compose.test.yml down
```

## 总结

本API规范定义了VGO Admin Gateway模块化系统的完整API标准，包括：

1. **核心系统API** - 模块管理、配置管理、健康检查等
2. **模块开发标准** - 接口规范、响应格式、错误处理等
3. **最佳实践** - 认证授权、日志记录、性能监控等

遵循这些规范可以确保模块间的一致性和互操作性，提高系统的可维护性和可扩展性。