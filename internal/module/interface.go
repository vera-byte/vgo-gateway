package module

// Module 模块接口定义 (为了向后兼容，继承BaseModule)
type Module interface {
	BaseModule
}

// ModuleInfo 模块信息
type ModuleInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// ModuleConfig 模块配置
type ModuleConfig struct {
	Enabled bool                   `mapstructure:"enabled" json:"enabled"`
	Config  map[string]interface{} `mapstructure:"config" json:"config,omitempty"`
}