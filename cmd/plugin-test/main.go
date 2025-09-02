package main

import (
	"os"
	"path/filepath"

	"github.com/vera-byte/vgo-gateway/internal/plugin"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

// main 插件加载器测试程序
// 用于测试VKP插件的加载和管理功能
func main() {
	// 创建日志记录器
	logger, err := zap.NewDevelopment()
	if err != nil {
		vgokit.Log.Fatal("创建日志记录器失败", zap.Error(err))
	}
	defer logger.Sync()
	
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		logger.Fatal("获取工作目录失败", zap.Error(err))
	}
	
	// 创建插件目录
	pluginDir := filepath.Join(wd, "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		logger.Fatal("创建插件目录失败", zap.Error(err))
	}
	
	// 创建VKP插件加载器
	loader := plugin.NewVKPLoader(pluginDir, logger)
	
	// 测试加载IAM模块的VKP文件
	vkpPath := filepath.Join(wd, "modules", "iam", "iam.vkp")
	logger.Info("开始测试VKP插件加载", zap.String("vkp_path", vkpPath))
	
	// 检查VKP文件是否存在
	if _, err := os.Stat(vkpPath); os.IsNotExist(err) {
		logger.Error("VKP文件不存在", zap.String("path", vkpPath))
		logger.Info("请先运行 'cd modules/iam && ./build.sh package' 来生成VKP文件")
		os.Exit(1)
	}
	
	// 加载插件
	pluginInstance, err := loader.LoadPlugin(vkpPath)
	if err != nil {
		logger.Error("加载VKP插件失败", zap.Error(err))
		os.Exit(1)
	}
	
	// 显示插件信息
	logger.Info("插件加载成功",
		zap.String("name", pluginInstance.GetName()),
		zap.String("version", pluginInstance.GetVersion()),
		zap.String("description", pluginInstance.GetDescription()))
	
	// 列出已加载的插件
	loadedPlugins := loader.ListPlugins()
	logger.Info("已加载的插件列表", zap.Strings("plugins", loadedPlugins))
	
	// 获取插件元数据
	if metadata := pluginInstance.GetMetadata(); metadata != nil {
		logger.Info("插件元数据",
			zap.String("name", metadata.Name),
			zap.String("version", metadata.Version),
			zap.String("description", metadata.Description),
			zap.String("author", metadata.Author),
			zap.String("license", metadata.License),
			zap.Bool("standalone_support", metadata.Standalone))
	}
	
	// 测试插件健康检查
	health, err := pluginInstance.Health()
	if err != nil {
		logger.Warn("插件健康检查失败", zap.Error(err))
	} else {
		logger.Info("插件健康状态", zap.Any("health", health))
	}
	
	// 卸载插件
	logger.Info("开始卸载插件")
	if err := loader.UnloadPlugin(pluginInstance.GetName()); err != nil {
		logger.Error("卸载插件失败", zap.Error(err))
	} else {
		logger.Info("插件卸载成功")
	}
	
	// 再次列出已加载的插件
	loadedPlugins = loader.ListPlugins()
	logger.Info("卸载后的插件列表", zap.Strings("plugins", loadedPlugins))
	
	logger.Info("VKP插件加载器测试完成")
}