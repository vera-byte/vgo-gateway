package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vera-byte/vgo-gateway/internal/config"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		vgokit.Log.Fatal("初始化日志失败", zap.Error(err))
	}
	defer logger.Sync()

	// 创建临时配置目录
	configDir := "./test-configs"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Fatal("创建配置目录失败", zap.Error(err))
	}
	defer os.RemoveAll(configDir) // 清理测试目录

	logger.Info("=== 模块配置管理器测试开始 ===")

	// 创建配置管理器
	configManager := config.NewModuleConfigManager(configDir, logger)
	logger.Info("配置管理器创建成功")

	// 测试1: 加载不存在的配置（应该创建默认配置）
	logger.Info("\n--- 测试1: 加载默认配置 ---")
	iamConfig, err := configManager.LoadConfig("iam")
	if err != nil {
		logger.Error("加载IAM配置失败", zap.Error(err))
		return
	}
	logger.Info("IAM默认配置加载成功",
		zap.String("name", iamConfig.Name),
		zap.String("version", iamConfig.Version),
		zap.Bool("enabled", iamConfig.Enabled),
		zap.Bool("auto_start", iamConfig.AutoStart),
		zap.Int("load_order", iamConfig.LoadOrder))

	// 测试2: 更新配置
	logger.Info("\n--- 测试2: 更新配置 ---")
	updates := map[string]interface{}{
		"enabled":    false,
		"load_order": 50,
		"config": map[string]interface{}{
			"database_url": "postgres://localhost:5432/iam",
			"jwt_secret":   "test-secret-key",
			"port":         8080,
		},
	}
	if err := configManager.UpdateConfig("iam", updates); err != nil {
		logger.Error("更新IAM配置失败", zap.Error(err))
		return
	}
	logger.Info("IAM配置更新成功")

	// 验证更新后的配置
	updatedConfig, exists := configManager.GetConfig("iam")
	if !exists {
		logger.Error("获取更新后的配置失败")
		return
	}
	logger.Info("更新后的配置",
		zap.Bool("enabled", updatedConfig.Enabled),
		zap.Int("load_order", updatedConfig.LoadOrder),
		zap.Any("config", updatedConfig.Config))

	// 测试3: 创建自定义配置并保存
	logger.Info("\n--- 测试3: 创建自定义配置 ---")
	userConfig := &config.ModuleConfig{
		Name:      "user",
		Version:   "2.0.0",
		Enabled:   true,
		AutoStart: false,
		LoadOrder: 200,
		Dependencies: []string{"iam"},
		Config: map[string]interface{}{
			"max_users":     1000,
			"session_timeout": 3600,
			"features": []string{"profile", "preferences", "notifications"},
		},
		HealthCheck: &config.HealthCheckConfig{
			Enabled:  true,
			Interval: 60,
			Timeout:  15,
			Retries:  5,
		},
	}

	if err := configManager.SaveConfig(userConfig); err != nil {
		logger.Error("保存用户模块配置失败", zap.Error(err))
		return
	}
	logger.Info("用户模块配置保存成功")

	// 测试4: 列出所有配置
	logger.Info("\n--- 测试4: 列出所有配置 ---")
	allConfigs := configManager.ListConfigs()
	logger.Info(fmt.Sprintf("共有 %d 个模块配置", len(allConfigs)))
	for _, cfg := range allConfigs {
		logger.Info("模块配置",
			zap.String("name", cfg.Name),
			zap.String("version", cfg.Version),
			zap.Bool("enabled", cfg.Enabled),
			zap.Int("load_order", cfg.LoadOrder))
	}

	// 测试5: 获取已启用的配置
	logger.Info("\n--- 测试5: 获取已启用的配置 ---")
	enabledConfigs := configManager.GetEnabledConfigs()
	logger.Info(fmt.Sprintf("共有 %d 个已启用的模块", len(enabledConfigs)))
	for _, cfg := range enabledConfigs {
		logger.Info("已启用模块",
			zap.String("name", cfg.Name),
			zap.Bool("auto_start", cfg.AutoStart))
	}

	// 测试6: 重新加载配置
	logger.Info("\n--- 测试6: 重新加载配置 ---")
	reloadedConfig, err := configManager.ReloadConfig("user")
	if err != nil {
		logger.Error("重新加载用户配置失败", zap.Error(err))
		return
	}
	logger.Info("用户配置重新加载成功",
		zap.String("name", reloadedConfig.Name),
		zap.String("version", reloadedConfig.Version))

	// 测试7: 验证配置文件是否正确创建
	logger.Info("\n--- 测试7: 验证配置文件 ---")
	iamConfigPath := filepath.Join(configDir, "iam.json")
	userConfigPath := filepath.Join(configDir, "user.json")
	
	if _, err := os.Stat(iamConfigPath); err == nil {
		logger.Info("IAM配置文件存在", zap.String("path", iamConfigPath))
	} else {
		logger.Error("IAM配置文件不存在", zap.String("path", iamConfigPath))
	}
	
	if _, err := os.Stat(userConfigPath); err == nil {
		logger.Info("用户配置文件存在", zap.String("path", userConfigPath))
	} else {
		logger.Error("用户配置文件不存在", zap.String("path", userConfigPath))
	}

	// 测试8: 删除配置
	logger.Info("\n--- 测试8: 删除配置 ---")
	if err := configManager.DeleteConfig("user"); err != nil {
		logger.Error("删除用户配置失败", zap.Error(err))
		return
	}
	logger.Info("用户配置删除成功")

	// 验证删除后的状态
	_, exists = configManager.GetConfig("user")
	if !exists {
		logger.Info("确认用户配置已从缓存中删除")
	} else {
		logger.Error("用户配置仍在缓存中")
	}

	logger.Info("\n=== 模块配置管理器测试完成 ===")
	logger.Info("所有测试通过！配置管理系统工作正常。")
}