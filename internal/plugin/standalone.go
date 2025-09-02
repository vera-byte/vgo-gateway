package plugin

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StandaloneRunner 独立运行器
// 支持模块的独立运行和测试
type StandaloneRunner struct {
	// plugin 插件实例
	plugin Plugin
	
	// logger 日志记录器
	logger *zap.Logger
	
	// server HTTP服务器
	server *http.Server
}

// NewStandaloneRunner 创建新的独立运行器
// plugin: 插件实例
// logger: 日志记录器
// 返回: 独立运行器实例
func NewStandaloneRunner(plugin Plugin, logger *zap.Logger) *StandaloneRunner {
	return &StandaloneRunner{
		plugin: plugin,
		logger: logger,
	}
}

// Run 运行插件
// ctx: 上下文
// port: 监听端口
// 返回: 错误信息
func (r *StandaloneRunner) Run(ctx context.Context, port int) error {
	// 检查插件是否支持独立运行
	if !r.plugin.CanRunStandalone() {
		return fmt.Errorf("plugin '%s' does not support standalone mode", r.plugin.GetName())
	}
	
	// 初始化插件
	if err := r.plugin.Initialize(ctx, r.logger, nil); err != nil {
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}
	
	// 创建Gin引擎
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	
	// 添加CORS中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	// 注册插件路由
	if err := r.plugin.RegisterRoutes(router); err != nil {
		return fmt.Errorf("failed to register plugin routes: %w", err)
	}
	
	// 添加健康检查路由
	router.GET("/health", r.healthHandler)
	
	// 添加插件信息路由
	router.GET("/info", r.infoHandler)
	
	// 创建HTTP服务器
	addr := fmt.Sprintf(":%d", port)
	r.server = &http.Server{
		Addr:    addr,
		Handler: router,
	}
	
	// 启动服务器
	go func() {
		r.logger.Info("Starting standalone plugin server", 
			zap.String("plugin", r.plugin.GetName()),
			zap.String("addr", addr))
		
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error("Failed to start server", zap.Error(err))
		}
	}()
	
	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case <-ctx.Done():
		r.logger.Info("Context cancelled, shutting down...")
	case sig := <-sigChan:
		r.logger.Info("Received signal, shutting down...", zap.String("signal", sig.String()))
	}
	
	// 优雅关闭
	return r.shutdown()
}

// shutdown 关闭服务器和插件
// 返回: 错误信息
func (r *StandaloneRunner) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 关闭HTTP服务器
	if r.server != nil {
		if err := r.server.Shutdown(ctx); err != nil {
			r.logger.Error("Failed to shutdown server", zap.Error(err))
		}
	}
	
	// 关闭插件
	if err := r.plugin.Shutdown(ctx); err != nil {
		r.logger.Error("Failed to shutdown plugin", zap.Error(err))
		return err
	}
	
	r.logger.Info("Standalone plugin server shutdown complete")
	return nil
}

// healthHandler 健康检查处理器
func (r *StandaloneRunner) healthHandler(c *gin.Context) {
	health, err := r.plugin.Health()
	if err != nil {
		c.JSON(500, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}
	
	c.JSON(200, health)
}

// infoHandler 插件信息处理器
func (r *StandaloneRunner) infoHandler(c *gin.Context) {
	metadata := r.plugin.GetMetadata()
	info := gin.H{
		"name":        r.plugin.GetName(),
		"version":     r.plugin.GetVersion(),
		"description": r.plugin.GetDescription(),
		"standalone":  r.plugin.CanRunStandalone(),
	}
	
	if metadata != nil {
		info["metadata"] = metadata
	}
	
	c.JSON(200, info)
}

// RunStandaloneFromArgs 从命令行参数运行独立模式
// plugin: 插件实例
// logger: 日志记录器
// args: 命令行参数
// 返回: 错误信息
func RunStandaloneFromArgs(plugin Plugin, logger *zap.Logger, args []string) error {
	// 解析命令行参数
	port := 8080 // 默认端口
	
	for i, arg := range args {
		switch arg {
		case "--port":
			if i+1 < len(args) {
				if p, err := strconv.Atoi(args[i+1]); err == nil {
					port = p
				}
			}
		case "--help", "-h":
			fmt.Printf("Usage: %s [options]\n", os.Args[0])
			fmt.Println("Options:")
			fmt.Println("  --port <port>    Listen port (default: 8080)")
			fmt.Println("  --help, -h       Show this help message")
			fmt.Println("  --metadata       Show plugin metadata")
			return nil
		case "--metadata":
			metadata := plugin.GetMetadata()
			if metadata != nil {
				fmt.Printf("Name: %s\n", metadata.Name)
				fmt.Printf("Version: %s\n", metadata.Version)
				fmt.Printf("Description: %s\n", metadata.Description)
				fmt.Printf("Author: %s\n", metadata.Author)
				fmt.Printf("Standalone: %t\n", metadata.Standalone)
			} else {
				fmt.Printf("Name: %s\n", plugin.GetName())
				fmt.Printf("Version: %s\n", plugin.GetVersion())
				fmt.Printf("Description: %s\n", plugin.GetDescription())
			}
			return nil
		}
	}
	
	// 创建并运行独立运行器
	runner := NewStandaloneRunner(plugin, logger)
	return runner.Run(context.Background(), port)
}