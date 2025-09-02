package main

import (
	"context"
	"flag"
	"os"
	"strconv"

	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"

	"github.com/vera-byte/vgo-gateway/internal/plugin"
	"github.com/vera-byte/vgo-gateway/modules/iam"
)

// main IAM模块独立运行入口
// 支持独立运行、测试和打包为VKP文件
func main() {
	// 解析命令行参数
	var (
		mode     = flag.String("mode", "standalone", "运行模式: standalone, gateway, metadata")
		port     = flag.Int("port", 8080, "监听端口")
		help     = flag.Bool("help", false, "显示帮助信息")
		metadata = flag.Bool("metadata", false, "显示插件元数据")
	)
	flag.Parse()

	// 显示帮助信息
	if *help {
		showHelp()
		return
	}

	// 创建IAM模块实例
	iamModule := iam.NewIAMModule()

	// 根据模式执行不同操作
	switch *mode {
	case "metadata":
		showMetadata(iamModule)
	case "standalone":
		runStandalone(iamModule, vgokit.Log.Logger, *port)
	case "gateway":
		runGatewayMode(iamModule, vgokit.Log.Logger)
	default:
		vgokit.Log.Error("Unknown mode", zap.String("mode", *mode))
		os.Exit(1)
	}

	// 处理旧式命令行参数（兼容VKP加载器）
	if *metadata || len(os.Args) > 1 && os.Args[1] == "--metadata" {
		showMetadata(iamModule)
		return
	}

	// 处理端口参数
	if len(os.Args) > 2 && os.Args[1] == "--port" {
		if p, err := strconv.Atoi(os.Args[2]); err == nil {
			*port = p
		}
	}

	// 默认独立运行模式
	runStandalone(iamModule, vgokit.Log.Logger, *port)
}

// showHelp 显示帮助信息
func showHelp() {
	vgokit.Log.Info("VGO Gateway")
	vgokit.Log.Info("Usage", zap.String("command", os.Args[0]+" [options]"))
	vgokit.Log.Info("Options:")
	vgokit.Log.Info("  -mode string        运行模式: standalone, gateway, metadata (default \"standalone\")")
	vgokit.Log.Info("  -port int           监听端口 (default 8080)")
	vgokit.Log.Info("  -help               显示此帮助信息")
	vgokit.Log.Info("  -metadata           显示插件元数据")
	vgokit.Log.Info("Examples:")
	vgokit.Log.Info("Standalone mode", zap.String("example", os.Args[0]+" -mode standalone -port 8080"))
	vgokit.Log.Info("Metadata mode", zap.String("example", os.Args[0]+" -metadata"))
	vgokit.Log.Info("VKP compatible", zap.String("example", os.Args[0]+" --metadata"))
}

// showMetadata 显示插件元数据
// module: IAM模块实例
func showMetadata(module *iam.IAMModule) {
	metadata := module.GetMetadata()
	if metadata != nil {
		vgokit.Log.Info("Plugin Metadata",
			zap.String("name", metadata.Name),
			zap.String("version", metadata.Version),
			zap.String("description", metadata.Description),
			zap.String("author", metadata.Author),
			zap.String("license", metadata.License),
			zap.String("api_version", metadata.APIVersion),
			zap.String("min_gateway_version", metadata.MinGatewayVersion),
			zap.Bool("standalone", metadata.Standalone),
			zap.Any("dependencies", metadata.Dependencies))
	} else {
		vgokit.Log.Info("Module Metadata",
			zap.String("name", module.GetName()),
			zap.String("version", module.GetVersion()),
			zap.String("description", module.GetDescription()),
			zap.Bool("standalone", module.CanRunStandalone()))
	}
}

// runStandalone 独立运行模式
// module: IAM模块实例
// logger: 日志记录器
// port: 监听端口
func runStandalone(module *iam.IAMModule, logger *zap.Logger, port int) {
	logger.Info("Starting IAM module in standalone mode",
		zap.String("name", module.GetName()),
		zap.String("version", module.GetVersion()),
		zap.Int("port", port))

	ctx := context.Background()
	if err := module.RunStandalone(ctx, port); err != nil {
		logger.Error("Failed to run standalone", zap.Error(err))
		os.Exit(1)
	}
}

// runGatewayMode 网关模式
// module: IAM模块实例
// logger: 日志记录器
func runGatewayMode(module *iam.IAMModule, logger *zap.Logger) {
	logger.Info("Starting IAM module in gateway mode",
		zap.String("name", module.GetName()),
		zap.String("version", module.GetVersion()))

	// 在网关模式下，模块作为子进程运行
	// 这里可以实现与主网关进程的通信逻辑
	// 目前简化为独立运行模式
	runStandalone(module, logger, 8080)
}

// NewPlugin 插件工厂函数（用于Go plugin加载）
// 返回: 插件实例
func NewPlugin() plugin.Plugin {
	return iam.NewIAMModule()
}
