package example

import (
	"github.com/vera-byte/vgo-gateway/internal/module"
)

// ExampleModuleFactory 示例模块工厂
type ExampleModuleFactory struct{}

// NewExampleModuleFactory 创建新的示例模块工厂
// 返回值: *ExampleModuleFactory 示例模块工厂实例
func NewExampleModuleFactory() *ExampleModuleFactory {
	return &ExampleModuleFactory{}
}

// CreateModule 创建示例模块实例
// 返回值: module.BaseModule 模块实例, error 错误信息
func (f *ExampleModuleFactory) CreateModule() (module.BaseModule, error) {
	return NewExampleModule(), nil
}

// ModuleType 获取模块类型
// 返回值: string 模块类型
func (f *ExampleModuleFactory) ModuleType() string {
	return "example"
}