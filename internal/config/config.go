package config

import (
	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server    ServerConfig           `mapstructure:"server" json:"server"`
	IAM       IAMConfig              `mapstructure:"iam" json:"iam"`
	JWT       JWTConfig              `mapstructure:"jwt" json:"jwt"`
	Log       LogConfig              `mapstructure:"log" json:"log"`
	RateLimit RateLimitConfig        `mapstructure:"ratelimit" json:"ratelimit"`
	Modules   map[string]interface{} `mapstructure:"modules" json:"modules"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `mapstructure:"port" json:"port"`
	Mode string `mapstructure:"mode" json:"mode"`
	Host string `mapstructure:"host" json:"host"`
}

// IAMConfig IAM服务配置
type IAMConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  int    `mapstructure:"timeout"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled    bool   `mapstructure:"enabled" json:"enabled"`
	Type       string `mapstructure:"type" json:"type"`         // redis 或 memory
	RedisAddr  string `mapstructure:"redis_addr" json:"redis_addr"`
	RedisDB    int    `mapstructure:"redis_db" json:"redis_db"`
	Rate       int    `mapstructure:"rate" json:"rate"`           // 每秒允许的请求数
	Burst      int    `mapstructure:"burst" json:"burst"`         // 突发请求数
	Expiration int    `mapstructure:"expiration" json:"expiration"` // 过期时间（秒）
}

// Load 加载配置文件
// 返回值: *Config 配置对象, error 错误信息
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("iam.endpoint", "localhost:9090")
	viper.SetDefault("iam.timeout", 30)
	viper.SetDefault("jwt.secret", "vgo-admin-gateway-secret")
	viper.SetDefault("jwt.expiration", 3600)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("ratelimit.enabled", false)
	viper.SetDefault("ratelimit.type", "memory")
	viper.SetDefault("ratelimit.rate", 100)
	viper.SetDefault("ratelimit.burst", 200)
	viper.SetDefault("ratelimit.expiration", 60)

	// 读取环境变量
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}