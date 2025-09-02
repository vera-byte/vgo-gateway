package iam

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/vera-byte/vgo-gateway/internal/module"
	"github.com/vera-byte/vgo-gateway/internal/plugin"
)

// IAMModuleAdapter BaseModule接口适配器
type IAMModuleAdapter struct {
	module *IAMModule
}

// Name 获取模块名称
func (a *IAMModuleAdapter) Name() string {
	return a.module.Name()
}

// Version 获取模块版本
func (a *IAMModuleAdapter) Version() string {
	return a.module.Version()
}

// Description 获取模块描述
func (a *IAMModuleAdapter) Description() string {
	return a.module.Description()
}

// Initialize 初始化模块
func (a *IAMModuleAdapter) Initialize(ctx context.Context, config interface{}, logger *zap.Logger) error {
	return a.module.InitializeForModule(ctx, config, logger)
}

// RegisterRoutes 注册模块路由
func (a *IAMModuleAdapter) RegisterRoutes(router *gin.RouterGroup, logger *zap.Logger) error {
	return a.module.registerRoutesGroup(router, logger)
}

// HealthCheck 健康检查
func (a *IAMModuleAdapter) HealthCheck(ctx context.Context) error {
	return a.module.HealthCheck(ctx)
}

// Shutdown 关闭模块
func (a *IAMModuleAdapter) Shutdown(ctx context.Context) error {
	return a.module.Shutdown(ctx)
}

// IAMModuleFactory IAM模块工厂
type IAMModuleFactory struct{}

// NewIAMModuleFactory 创建新的IAM模块工厂
// 返回值: *IAMModuleFactory IAM模块工厂实例
func NewIAMModuleFactory() *IAMModuleFactory {
	return &IAMModuleFactory{}
}

// CreateModule 创建IAM模块实例
// 返回值: module.BaseModule 模块实例, error 错误信息
func (f *IAMModuleFactory) CreateModule() (module.BaseModule, error) {
	iamModule := NewIAMModule()
	return &IAMModuleAdapter{module: iamModule}, nil
}

// CreatePlugin 创建IAM插件实例
// 返回值: plugin.Plugin 插件实例, error 错误信息
func (f *IAMModuleFactory) CreatePlugin() (plugin.Plugin, error) {
	return NewIAMModule(), nil
}

// ModuleType 获取模块类型
// 返回值: string 模块类型
func (f *IAMModuleFactory) ModuleType() string {
	return "iam"
}