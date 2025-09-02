# 模块开发示例：创建用户管理模块

本文档将通过创建一个完整的用户管理模块来演示如何开发VGO Admin Gateway模块。

## 1. 项目初始化

### 创建模块目录结构

```bash
# 创建模块根目录
mkdir -p modules/user
cd modules/user

# 初始化Go模块
go mod init user

# 创建目录结构
mkdir -p cmd/server
mkdir -p internal/handler
mkdir -p internal/service
mkdir -p internal/model
mkdir -p internal/config
mkdir -p internal/repository
```

### 创建go.mod文件

```go
// go.mod
module user

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/vera-byte/vgo-gateway v0.0.0
    go.uber.org/zap v1.24.0
    gorm.io/gorm v1.25.0
    gorm.io/driver/postgres v1.5.0
)
```

## 2. 定义数据模型

### 用户模型

```go
// internal/model/user.go
package model

import (
    "time"
    "gorm.io/gorm"
)

// User 用户模型
type User struct {
    ID        uint           `json:"id" gorm:"primarykey"`
    Username  string         `json:"username" gorm:"uniqueIndex;not null"`
    Email     string         `json:"email" gorm:"uniqueIndex;not null"`
    Password  string         `json:"-" gorm:"not null"` // 不在JSON中显示密码
    FirstName string         `json:"first_name"`
    LastName  string         `json:"last_name"`
    Avatar    string         `json:"avatar"`
    Status    UserStatus     `json:"status" gorm:"default:active"`
    LastLogin *time.Time     `json:"last_login"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// UserStatus 用户状态枚举
type UserStatus string

const (
    UserStatusActive   UserStatus = "active"
    UserStatusInactive UserStatus = "inactive"
    UserStatusBlocked  UserStatus = "blocked"
)

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
    Username  string `json:"username" binding:"required,min=3,max=50"`
    Email     string `json:"email" binding:"required,email"`
    Password  string `json:"password" binding:"required,min=6"`
    FirstName string `json:"first_name" binding:"max=50"`
    LastName  string `json:"last_name" binding:"max=50"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
    Email     *string     `json:"email,omitempty" binding:"omitempty,email"`
    FirstName *string     `json:"first_name,omitempty" binding:"omitempty,max=50"`
    LastName  *string     `json:"last_name,omitempty" binding:"omitempty,max=50"`
    Avatar    *string     `json:"avatar,omitempty"`
    Status    *UserStatus `json:"status,omitempty"`
}

// UserResponse 用户响应
type UserResponse struct {
    ID        uint       `json:"id"`
    Username  string     `json:"username"`
    Email     string     `json:"email"`
    FirstName string     `json:"first_name"`
    LastName  string     `json:"last_name"`
    Avatar    string     `json:"avatar"`
    Status    UserStatus `json:"status"`
    LastLogin *time.Time `json:"last_login"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}

// ToResponse 转换为响应格式
func (u *User) ToResponse() *UserResponse {
    return &UserResponse{
        ID:        u.ID,
        Username:  u.Username,
        Email:     u.Email,
        FirstName: u.FirstName,
        LastName:  u.LastName,
        Avatar:    u.Avatar,
        Status:    u.Status,
        LastLogin: u.LastLogin,
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
    }
}
```

## 3. 实现数据访问层

### 用户仓储

```go
// internal/repository/user.go
package repository

import (
    "user/internal/model"
    "gorm.io/gorm"
)

// UserRepository 用户仓储接口
type UserRepository interface {
    Create(user *model.User) error
    GetByID(id uint) (*model.User, error)
    GetByUsername(username string) (*model.User, error)
    GetByEmail(email string) (*model.User, error)
    Update(user *model.User) error
    Delete(id uint) error
    List(offset, limit int) ([]*model.User, int64, error)
    Search(keyword string, offset, limit int) ([]*model.User, int64, error)
}

// userRepository 用户仓储实现
type userRepository struct {
    db *gorm.DB
}

// NewUserRepository 创建用户仓储实例
// db: 数据库连接
// 返回: 用户仓储接口
func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepository{db: db}
}

// Create 创建用户
// user: 用户模型
// 返回: 错误信息
func (r *userRepository) Create(user *model.User) error {
    return r.db.Create(user).Error
}

// GetByID 根据ID获取用户
// id: 用户ID
// 返回: 用户模型和错误信息
func (r *userRepository) GetByID(id uint) (*model.User, error) {
    var user model.User
    err := r.db.First(&user, id).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// GetByUsername 根据用户名获取用户
// username: 用户名
// 返回: 用户模型和错误信息
func (r *userRepository) GetByUsername(username string) (*model.User, error) {
    var user model.User
    err := r.db.Where("username = ?", username).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// GetByEmail 根据邮箱获取用户
// email: 邮箱地址
// 返回: 用户模型和错误信息
func (r *userRepository) GetByEmail(email string) (*model.User, error) {
    var user model.User
    err := r.db.Where("email = ?", email).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// Update 更新用户
// user: 用户模型
// 返回: 错误信息
func (r *userRepository) Update(user *model.User) error {
    return r.db.Save(user).Error
}

// Delete 删除用户
// id: 用户ID
// 返回: 错误信息
func (r *userRepository) Delete(id uint) error {
    return r.db.Delete(&model.User{}, id).Error
}

// List 获取用户列表
// offset: 偏移量
// limit: 限制数量
// 返回: 用户列表、总数和错误信息
func (r *userRepository) List(offset, limit int) ([]*model.User, int64, error) {
    var users []*model.User
    var total int64
    
    // 获取总数
    if err := r.db.Model(&model.User{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    // 获取分页数据
    err := r.db.Offset(offset).Limit(limit).Find(&users).Error
    return users, total, err
}

// Search 搜索用户
// keyword: 搜索关键词
// offset: 偏移量
// limit: 限制数量
// 返回: 用户列表、总数和错误信息
func (r *userRepository) Search(keyword string, offset, limit int) ([]*model.User, int64, error) {
    var users []*model.User
    var total int64
    
    query := r.db.Model(&model.User{}).Where(
        "username ILIKE ? OR email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
        "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
    )
    
    // 获取总数
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    // 获取分页数据
    err := query.Offset(offset).Limit(limit).Find(&users).Error
    return users, total, err
}
```

## 4. 实现业务逻辑层

### 用户服务

```go
// internal/service/user.go
package service

import (
    "errors"
    "time"
    "golang.org/x/crypto/bcrypt"
    "user/internal/model"
    "user/internal/repository"
    "go.uber.org/zap"
)

// UserService 用户服务接口
type UserService interface {
    CreateUser(req *model.CreateUserRequest) (*model.UserResponse, error)
    GetUser(id uint) (*model.UserResponse, error)
    UpdateUser(id uint, req *model.UpdateUserRequest) (*model.UserResponse, error)
    DeleteUser(id uint) error
    ListUsers(page, pageSize int) ([]*model.UserResponse, int64, error)
    SearchUsers(keyword string, page, pageSize int) ([]*model.UserResponse, int64, error)
    ValidatePassword(username, password string) (*model.UserResponse, error)
}

// userService 用户服务实现
type userService struct {
    userRepo repository.UserRepository
    logger   *zap.Logger
}

// NewUserService 创建用户服务实例
// userRepo: 用户仓储
// logger: 日志记录器
// 返回: 用户服务接口
func NewUserService(userRepo repository.UserRepository, logger *zap.Logger) UserService {
    return &userService{
        userRepo: userRepo,
        logger:   logger,
    }
}

// CreateUser 创建用户
// req: 创建用户请求
// 返回: 用户响应和错误信息
func (s *userService) CreateUser(req *model.CreateUserRequest) (*model.UserResponse, error) {
    // 检查用户名是否已存在
    if _, err := s.userRepo.GetByUsername(req.Username); err == nil {
        return nil, errors.New("用户名已存在")
    }
    
    // 检查邮箱是否已存在
    if _, err := s.userRepo.GetByEmail(req.Email); err == nil {
        return nil, errors.New("邮箱已存在")
    }
    
    // 加密密码
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        s.logger.Error("密码加密失败", zap.Error(err))
        return nil, errors.New("密码加密失败")
    }
    
    // 创建用户
    user := &model.User{
        Username:  req.Username,
        Email:     req.Email,
        Password:  string(hashedPassword),
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Status:    model.UserStatusActive,
    }
    
    if err := s.userRepo.Create(user); err != nil {
        s.logger.Error("创建用户失败", zap.Error(err))
        return nil, errors.New("创建用户失败")
    }
    
    s.logger.Info("用户创建成功", zap.String("username", user.Username))
    return user.ToResponse(), nil
}

// GetUser 获取用户
// id: 用户ID
// 返回: 用户响应和错误信息
func (s *userService) GetUser(id uint) (*model.UserResponse, error) {
    user, err := s.userRepo.GetByID(id)
    if err != nil {
        return nil, errors.New("用户不存在")
    }
    return user.ToResponse(), nil
}

// UpdateUser 更新用户
// id: 用户ID
// req: 更新用户请求
// 返回: 用户响应和错误信息
func (s *userService) UpdateUser(id uint, req *model.UpdateUserRequest) (*model.UserResponse, error) {
    user, err := s.userRepo.GetByID(id)
    if err != nil {
        return nil, errors.New("用户不存在")
    }
    
    // 更新字段
    if req.Email != nil {
        // 检查邮箱是否已被其他用户使用
        if existingUser, err := s.userRepo.GetByEmail(*req.Email); err == nil && existingUser.ID != id {
            return nil, errors.New("邮箱已被使用")
        }
        user.Email = *req.Email
    }
    
    if req.FirstName != nil {
        user.FirstName = *req.FirstName
    }
    
    if req.LastName != nil {
        user.LastName = *req.LastName
    }
    
    if req.Avatar != nil {
        user.Avatar = *req.Avatar
    }
    
    if req.Status != nil {
        user.Status = *req.Status
    }
    
    if err := s.userRepo.Update(user); err != nil {
        s.logger.Error("更新用户失败", zap.Error(err))
        return nil, errors.New("更新用户失败")
    }
    
    s.logger.Info("用户更新成功", zap.Uint("user_id", id))
    return user.ToResponse(), nil
}

// DeleteUser 删除用户
// id: 用户ID
// 返回: 错误信息
func (s *userService) DeleteUser(id uint) error {
    if _, err := s.userRepo.GetByID(id); err != nil {
        return errors.New("用户不存在")
    }
    
    if err := s.userRepo.Delete(id); err != nil {
        s.logger.Error("删除用户失败", zap.Error(err))
        return errors.New("删除用户失败")
    }
    
    s.logger.Info("用户删除成功", zap.Uint("user_id", id))
    return nil
}

// ListUsers 获取用户列表
// page: 页码
// pageSize: 每页大小
// 返回: 用户响应列表、总数和错误信息
func (s *userService) ListUsers(page, pageSize int) ([]*model.UserResponse, int64, error) {
    offset := (page - 1) * pageSize
    users, total, err := s.userRepo.List(offset, pageSize)
    if err != nil {
        return nil, 0, err
    }
    
    responses := make([]*model.UserResponse, len(users))
    for i, user := range users {
        responses[i] = user.ToResponse()
    }
    
    return responses, total, nil
}

// SearchUsers 搜索用户
// keyword: 搜索关键词
// page: 页码
// pageSize: 每页大小
// 返回: 用户响应列表、总数和错误信息
func (s *userService) SearchUsers(keyword string, page, pageSize int) ([]*model.UserResponse, int64, error) {
    offset := (page - 1) * pageSize
    users, total, err := s.userRepo.Search(keyword, offset, pageSize)
    if err != nil {
        return nil, 0, err
    }
    
    responses := make([]*model.UserResponse, len(users))
    for i, user := range users {
        responses[i] = user.ToResponse()
    }
    
    return responses, total, nil
}

// ValidatePassword 验证密码
// username: 用户名
// password: 密码
// 返回: 用户响应和错误信息
func (s *userService) ValidatePassword(username, password string) (*model.UserResponse, error) {
    user, err := s.userRepo.GetByUsername(username)
    if err != nil {
        return nil, errors.New("用户不存在")
    }
    
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
        return nil, errors.New("密码错误")
    }
    
    // 更新最后登录时间
    now := time.Now()
    user.LastLogin = &now
    s.userRepo.Update(user)
    
    return user.ToResponse(), nil
}
```

## 5. 实现HTTP处理器

### 用户处理器

```go
// internal/handler/user.go
package handler

import (
    "net/http"
    "strconv"
    "user/internal/model"
    "user/internal/service"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// UserHandler 用户处理器
type UserHandler struct {
    userService service.UserService
    logger      *zap.Logger
}

// NewUserHandler 创建用户处理器实例
// userService: 用户服务
// logger: 日志记录器
// 返回: 用户处理器指针
func NewUserHandler(userService service.UserService, logger *zap.Logger) *UserHandler {
    return &UserHandler{
        userService: userService,
        logger:      logger,
    }
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
    var req model.CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    user, err := h.userService.CreateUser(&req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{"data": user})
}

// GetUser 获取用户
func (h *UserHandler) GetUser(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
        return
    }
    
    user, err := h.userService.GetUser(uint(id))
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": user})
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
        return
    }
    
    var req model.UpdateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    user, err := h.userService.UpdateUser(uint(id), &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"data": user})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
        return
    }
    
    if err := h.userService.DeleteUser(uint(id)); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "用户删除成功"})
}

// ListUsers 获取用户列表
func (h *UserHandler) ListUsers(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
    
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 10
    }
    
    users, total, err := h.userService.ListUsers(page, pageSize)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": users,
        "pagination": gin.H{
            "page":       page,
            "page_size":  pageSize,
            "total":      total,
            "total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
        },
    })
}

// SearchUsers 搜索用户
func (h *UserHandler) SearchUsers(c *gin.Context) {
    keyword := c.Query("keyword")
    if keyword == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "搜索关键词不能为空"})
        return
    }
    
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
    
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 10
    }
    
    users, total, err := h.userService.SearchUsers(keyword, page, pageSize)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": users,
        "pagination": gin.H{
            "page":       page,
            "page_size":  pageSize,
            "total":      total,
            "total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
        },
    })
}
```

## 6. 实现模块主程序

### 主程序入口

```go
// cmd/server/main.go
package main

import (
    "fmt"
    "log"
    "os"
    "strconv"
    "user/internal/config"
    "user/internal/handler"
    "user/internal/model"
    "user/internal/repository"
    "user/internal/service"
    
    "github.com/gin-gonic/gin"
    "github.com/vera-byte/vgo-gateway/internal/module"
    "go.uber.org/zap"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

// UserModule 用户模块
type UserModule struct {
    db          *gorm.DB
    userHandler *handler.UserHandler
    logger      *zap.Logger
    config      *config.Config
}

// Name 返回模块名称
func (m *UserModule) Name() string {
    return "user"
}

// Version 返回模块版本
func (m *UserModule) Version() string {
    return "1.0.0"
}

// Description 返回模块描述
func (m *UserModule) Description() string {
    return "用户管理模块"
}

// Initialize 初始化模块
func (m *UserModule) Initialize() error {
    // 初始化日志
    logger, err := zap.NewProduction()
    if err != nil {
        return fmt.Errorf("初始化日志失败: %w", err)
    }
    m.logger = logger
    
    // 加载配置
    m.config = config.Load()
    
    // 初始化数据库
    db, err := gorm.Open(postgres.Open(m.config.DatabaseURL), &gorm.Config{})
    if err != nil {
        return fmt.Errorf("连接数据库失败: %w", err)
    }
    m.db = db
    
    // 自动迁移
    if err := db.AutoMigrate(&model.User{}); err != nil {
        return fmt.Errorf("数据库迁移失败: %w", err)
    }
    
    // 初始化依赖
    userRepo := repository.NewUserRepository(db)
    userService := service.NewUserService(userRepo, logger)
    m.userHandler = handler.NewUserHandler(userService, logger)
    
    m.logger.Info("用户模块初始化成功")
    return nil
}

// RegisterRoutes 注册路由
func (m *UserModule) RegisterRoutes(router *gin.RouterGroup) {
    userGroup := router.Group("/users")
    {
        userGroup.POST("", m.userHandler.CreateUser)
        userGroup.GET("/:id", m.userHandler.GetUser)
        userGroup.PUT("/:id", m.userHandler.UpdateUser)
        userGroup.DELETE("/:id", m.userHandler.DeleteUser)
        userGroup.GET("", m.userHandler.ListUsers)
        userGroup.GET("/search", m.userHandler.SearchUsers)
    }
    
    m.logger.Info("用户模块路由注册成功")
}

// HealthCheck 健康检查
func (m *UserModule) HealthCheck() error {
    // 检查数据库连接
    sqlDB, err := m.db.DB()
    if err != nil {
        return err
    }
    return sqlDB.Ping()
}

// Shutdown 关闭模块
func (m *UserModule) Shutdown() error {
    if m.db != nil {
        sqlDB, err := m.db.DB()
        if err == nil {
            sqlDB.Close()
        }
    }
    
    if m.logger != nil {
        m.logger.Sync()
    }
    
    return nil
}

// NewPlugin 插件入口函数
func NewPlugin() module.Module {
    return &UserModule{}
}

// main 主函数（用于独立运行）
func main() {
    if len(os.Args) > 1 && os.Args[1] == "standalone" {
        // 独立运行模式
        runStandalone()
    } else {
        // 插件模式（不应该直接运行）
        log.Fatal("此程序应作为插件运行")
    }
}

// runStandalone 独立运行模式
func runStandalone() {
    // 解析命令行参数
    port := 8080
    for i, arg := range os.Args {
        if arg == "--port" && i+1 < len(os.Args) {
            if p, err := strconv.Atoi(os.Args[i+1]); err == nil {
                port = p
            }
        }
    }
    
    // 创建模块实例
    userModule := &UserModule{}
    
    // 初始化模块
    if err := userModule.Initialize(); err != nil {
        log.Fatal("模块初始化失败:", err)
    }
    defer userModule.Shutdown()
    
    // 创建Gin路由
    gin.SetMode(gin.ReleaseMode)
    router := gin.New()
    router.Use(gin.Logger(), gin.Recovery())
    
    // 注册路由
    api := router.Group("/api/v1")
    userModule.RegisterRoutes(api)
    
    // 健康检查端点
    router.GET("/health", func(c *gin.Context) {
        if err := userModule.HealthCheck(); err != nil {
            c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"status": "healthy"})
    })
    
    // 启动服务器
    addr := fmt.Sprintf(":%d", port)
    log.Printf("用户模块独立运行在端口 %d", port)
    log.Fatal(router.Run(addr))
}
```

## 7. 配置管理

### 配置结构

```go
// internal/config/config.go
package config

import (
    "os"
)

// Config 配置结构
type Config struct {
    DatabaseURL string
    LogLevel    string
    Port        int
}

// Load 加载配置
// 返回: 配置实例
func Load() *Config {
    return &Config{
        DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/user_db?sslmode=disable"),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        Port:        8080,
    }
}

// getEnv 获取环境变量
// key: 环境变量键
// defaultValue: 默认值
// 返回: 环境变量值
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

## 8. 构建脚本

### build.sh

```bash
#!/bin/bash
# build.sh

set -e

MODULE_NAME="user"
MODULE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$MODULE_DIR/build"
OUTPUT_DIR="$MODULE_DIR"

case "$1" in
    "build")
        echo "构建 $MODULE_NAME 模块..."
        mkdir -p "$BUILD_DIR"
        
        # 构建插件
        go build -buildmode=plugin -o "$BUILD_DIR/plugin" ./cmd/server
        echo "插件构建完成: $BUILD_DIR/plugin"
        ;;
        
    "package")
        echo "打包 $MODULE_NAME 模块..."
        ./build.sh build
        
        # 创建元数据文件
        cat > "$BUILD_DIR/metadata.json" << EOF
{
    "name": "$MODULE_NAME",
    "version": "1.0.0",
    "description": "用户管理模块",
    "author": "VGO Team",
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
    "description": "用户管理模块",
    "config_schema": {
        "database_url": {
            "type": "string",
            "description": "数据库连接URL",
            "required": true
        },
        "log_level": {
            "type": "string",
            "description": "日志级别",
            "default": "info",
            "enum": ["debug", "info", "warn", "error"]
        }
    }
}
EOF
        
        # 打包为VKP文件
        cd "$BUILD_DIR"
        tar -czf "$OUTPUT_DIR/$MODULE_NAME.vkp" .
        echo "VKP包已创建: $OUTPUT_DIR/$MODULE_NAME.vkp"
        
        # 清理构建目录
        cd "$MODULE_DIR"
        rm -rf "$BUILD_DIR"
        ;;
        
    "run")
        echo "独立运行 $MODULE_NAME 模块..."
        shift # 移除第一个参数
        go run ./cmd/server/main.go standalone "$@"
        ;;
        
    "test")
        echo "运行 $MODULE_NAME 模块测试..."
        go test ./... -v
        ;;
        
    "clean")
        echo "清理 $MODULE_NAME 模块构建文件..."
        rm -rf "$BUILD_DIR"
        rm -f "$MODULE_NAME.vkp"
        echo "清理完成"
        ;;
        
    *)
        echo "用法: $0 {build|package|run|test|clean}"
        echo "  build   - 构建模块"
        echo "  package - 打包为VKP文件"
        echo "  run     - 独立运行模块"
        echo "  test    - 运行测试"
        echo "  clean   - 清理构建文件"
        exit 1
        ;;
esac
```

## 9. 测试和验证

### 独立运行测试

```bash
# 设置数据库环境变量
export DATABASE_URL="postgres://username:password@localhost:5432/user_db?sslmode=disable"

# 独立运行模块
./build.sh run --port 8080
```

### API测试

```bash
# 创建用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User"
  }'

# 获取用户列表
curl http://localhost:8080/api/v1/users

# 获取特定用户
curl http://localhost:8080/api/v1/users/1

# 搜索用户
curl "http://localhost:8080/api/v1/users/search?keyword=test"

# 健康检查
curl http://localhost:8080/health
```

### 打包和部署

```bash
# 打包模块
./build.sh package

# 验证VKP包
tar -tzf user.vkp

# 将VKP包复制到网关的plugins目录
cp user.vkp ../../plugins/
```

## 10. 总结

通过这个完整的示例，我们展示了如何：

1. **设计模块架构** - 使用分层架构（Handler -> Service -> Repository）
2. **实现业务逻辑** - 完整的用户管理功能
3. **数据库集成** - 使用GORM进行数据访问
4. **API设计** - RESTful API设计
5. **配置管理** - 环境变量配置
6. **独立运行** - 支持开发调试
7. **打包部署** - VKP格式打包
8. **测试验证** - API测试和健康检查

这个示例为开发其他模块提供了完整的参考模板。