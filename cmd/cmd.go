package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vera-byte/vgo-gateway/internal/api"
	"github.com/vera-byte/vgo-gateway/internal/config"
	"github.com/vera-byte/vgo-gateway/internal/middleware"
	"github.com/vera-byte/vgo-gateway/internal/module"
	"github.com/vera-byte/vgo-gateway/internal/plugin"
	"github.com/vera-byte/vgo-gateway/modules/example"
	"github.com/vera-byte/vgo-gateway/modules/iam"

	"github.com/gin-gonic/gin"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// RootCmd 根命令
var RootCmd = &cobra.Command{
	Use:   "vgo-gateway",
	Short: "VGO Gateway Server",
	Long:  `VGO Gateway is a modular API gateway with plugin support.`,
}

// serverCmd 服务器启动命令
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the gateway server",
	Long:  `Start the VGO Gateway server with all configured modules and plugins.`,
	Run:   runServer,
}

func init() {
	// 添加子命令
	RootCmd.AddCommand(serverCmd)
}

// runServer 启动服务器
// cmd: cobra命令实例
// args: 命令行参数
func runServer(cmd *cobra.Command, args []string) {
	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting VGO Admin Gateway...")

	// 加载配置
	logger.Info("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}
	logger.Info("Configuration loaded successfully")

	// 创建模块管理器
	logger.Info("Creating module manager...")
	moduleManager := module.NewManager(logger)

	// 创建插件管理器
	logger.Info("Creating plugin manager...")
	vpksDir := filepath.Join("plugins", "vpks")
	pluginManager := plugin.NewManager(logger, vpksDir)

	// 设置插件加载器
	pluginLoader := plugin.NewVKPLoader("plugins", logger)
	pluginManager.SetLoader(pluginLoader)

	// 注册IAM模块
	logger.Info("Registering IAM module...")
	iamFactory := iam.NewIAMModuleFactory()
	iamModule, err := iamFactory.CreateModule()
	if err != nil {
		logger.Fatal("Failed to create IAM module", zap.Error(err))
	}
	if err := moduleManager.RegisterModule("iam", iamModule); err != nil {
		logger.Fatal("Failed to register IAM module", zap.Error(err))
	}

	// 注册示例模块（可选）
	logger.Info("Registering Example module...")
	exampleModule := example.NewExampleModule()
	if err := moduleManager.RegisterModule("example", exampleModule); err != nil {
		logger.Fatal("Failed to register Example module", zap.Error(err))
	}

	// 初始化所有模块
	logger.Info("Initializing all modules...")
	ctx := context.Background()
	if err := moduleManager.InitializeAll(ctx, cfg.Modules); err != nil {
		logger.Fatal("Failed to initialize modules", zap.Error(err))
	}
	logger.Info("All modules initialized successfully")

	// 创建Gin引擎
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// 添加限流中间件
	if cfg.RateLimit.Enabled {
		logger.Info("Initializing rate limiter", zap.String("type", cfg.RateLimit.Type))
		// 初始化限流器
		rateLimitConfig := middleware.RateLimitConfig{
			Enabled:   cfg.RateLimit.Enabled,
			Type:      cfg.RateLimit.Type,
			Limit:     cfg.RateLimit.Rate,
			Window:    time.Duration(cfg.RateLimit.Expiration) * time.Second,
			Prefix:    "ratelimit:",
			RedisAddr: cfg.RateLimit.RedisAddr,
			RedisDB:   cfg.RateLimit.RedisDB,
		}
		rateLimiter, err := middleware.NewRateLimiter(rateLimitConfig)
		if err != nil {
			logger.Fatal("Failed to initialize rate limiter", zap.Error(err))
		}
		router.Use(middleware.RateLimitMiddleware(rateLimiter, middleware.DefaultKeyFunc))
		logger.Info("Rate limiter enabled")
	}

	// 健康检查路由
	router.GET("/health", func(c *gin.Context) {
		health := moduleManager.HealthCheck(ctx)
		if len(health) == 0 {
			c.JSON(http.StatusOK, gin.H{"status": "healthy", "modules": "none"})
		} else {
			allHealthy := true
			for _, err := range health {
				if err != nil {
					allHealthy = false
					break
				}
			}
			status := "healthy"
			if !allHealthy {
				status = "unhealthy"
			}
			c.JSON(http.StatusOK, gin.H{"status": status, "modules": health})
		}
	})

	// 注册模块路由
	apiGroup := router.Group("/api/v1")
	if err := moduleManager.RegisterRoutes(apiGroup, logger); err != nil {
		logger.Fatal("Failed to register module routes", zap.Error(err))
	}
	logger.Info("Module routes registered successfully")

	// 创建并注册插件API处理器
	logger.Info("Registering plugin API routes...")
	pluginHandler := api.NewPluginHandler(pluginManager, logger)
	pluginHandler.RegisterRoutes(router)
	logger.Info("Plugin API routes registered successfully")

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// 统计并输出路由和中间件信息
	printServerInfo(router, logger, pluginManager)

	// 启动服务器
	go func() {
		logger.Info("Starting server", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// 关闭所有模块
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := moduleManager.ShutdownAll(shutdownCtx); err != nil {
		logger.Error("Error shutting down modules", zap.Error(err))
	}

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// printServerInfo 输出服务器信息，包括路由数量和中间件信息
// router: Gin引擎实例
// logger: 日志记录器
func printServerInfo(router *gin.Engine, logger *zap.Logger, pluginManager *plugin.Manager) {
	// 获取详细路由信息
	routeDetails := getRouteDetails(router)

	// 输出服务器信息标题
	logger.Info("=== VGO Gateway Server Information ===")
	// 输出详细路由信息表格
	printRouteDetailsTable(routeDetails)

	// 输出插件信息表格
	printPluginTables(pluginManager)
}

// RouteStats 路由统计信息
type RouteStats struct {
	Total   int
	GET     int
	POST    int
	PUT     int
	DELETE  int
	PATCH   int
	OPTIONS int
	HEAD    int
}

// MiddlewareStats 中间件统计信息
type MiddlewareStats struct {
	Global int
	Group  int
	Total  int
	Names  []string
}

// RouteDetail 路由详细信息
type RouteDetail struct {
	Method  string
	Path    string
	Handler string
}

// countRoutes 统计路由数量
// router: Gin引擎实例
// 返回值: RouteStats 路由统计信息
func countRoutes(router *gin.Engine) RouteStats {
	routes := router.Routes()
	stats := RouteStats{}

	for _, route := range routes {
		stats.Total++
		switch route.Method {
		case "GET":
			stats.GET++
		case "POST":
			stats.POST++
		case "PUT":
			stats.PUT++
		case "DELETE":
			stats.DELETE++
		case "PATCH":
			stats.PATCH++
		case "OPTIONS":
			stats.OPTIONS++
		case "HEAD":
			stats.HEAD++
		}
	}

	return stats
}

// countMiddlewares 统计中间件数量
// router: Gin引擎实例
// 返回值: MiddlewareStats 中间件统计信息
func countMiddlewares(router *gin.Engine) MiddlewareStats {
	stats := MiddlewareStats{
		Names: []string{},
	}

	// 统计全局中间件（这里基于我们在main函数中添加的中间件）
	middlewareNames := []string{
		"gin.Logger",
		"gin.Recovery",
		"middleware.CORS",
	}

	// 统计路由组中间件（简化统计）
	// 由于Gin的内部结构限制，这里使用简化的统计方法
	stats.Group = 0 // 路由组级别的中间件数量（需要更复杂的逻辑来准确统计）

	// 检查配置以确定是否启用了限流中间件
	cfg, err := config.Load()
	if err == nil && cfg.RateLimit.Enabled {
		middlewareNames = append(middlewareNames, "middleware.RateLimit")
	}

	stats.Global = len(middlewareNames)
	stats.Total = stats.Global + stats.Group
	stats.Names = middlewareNames

	return stats
}

// getRouteDetails 获取路由详细信息
// router: Gin引擎实例
// 返回值: []RouteDetail 路由详细信息列表
func getRouteDetails(router *gin.Engine) []RouteDetail {
	routes := router.Routes()
	details := make([]RouteDetail, 0, len(routes))

	for _, route := range routes {
		detail := RouteDetail{
			Method:  route.Method,
			Path:    route.Path,
			Handler: route.Handler,
		}
		details = append(details, detail)
	}

	return details
}

// printRouteDetailsTable 输出路由详细信息表格
// details: 路由详细信息列表
func printRouteDetailsTable(details []RouteDetail) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Method", "Path", "Handler"})

	// 添加路由数据
	for _, detail := range details {
		// 截断过长的处理器名称
		handler := detail.Handler
		if len(handler) > 60 {
			handler = handler[:57] + "..."
		}
		table.Append([]string{detail.Method, detail.Path, handler})
	}

	// 配置表格样式
	table.SetBorder(false)
	table.SetCenterSeparator("|")
	table.SetColumnSeparator("|")
	table.SetRowSeparator("-")
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	fmt.Println("\n🛣️  Route Details:")
	table.Render()
}

// printPluginTables 输出插件信息表格
// pluginManager: 插件管理器实例
func printPluginTables(pluginManager *plugin.Manager) {
	// 获取插件列表
	plugins := pluginManager.ListPlugins()

	if len(plugins) == 0 {
		fmt.Println("\n🔌 Plugin Information:")
		fmt.Println("No plugins registered.")
		return
	}

	// 插件列表表格
	pluginTable := tablewriter.NewWriter(os.Stdout)
	pluginTable.SetHeader([]string{"Name", "Version", "Description", "Status"})

	for _, p := range plugins {
		name := p["name"].(string)
		version := p["version"].(string)
		description := "N/A"
		if desc, ok := p["description"].(string); ok && desc != "" {
			description = desc
		}
		if len(description) > 40 {
			description = description[:37] + "..."
		}

		status := "Loaded"
		if metadata, ok := p["metadata"]; !ok || metadata == nil {
			status = "Error"
		}

		pluginTable.Append([]string{name, version, description, status})
	}

	// 配置插件列表表格样式
	pluginTable.SetBorder(false)
	pluginTable.SetCenterSeparator("|")
	pluginTable.SetColumnSeparator("|")
	pluginTable.SetRowSeparator("-")
	pluginTable.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	pluginTable.SetAlignment(tablewriter.ALIGN_LEFT)

	fmt.Println("\n🔌 Plugin List:")
	pluginTable.Render()

	// 插件详细信息表格
	detailTable := tablewriter.NewWriter(os.Stdout)
	detailTable.SetHeader([]string{"Plugin", "Metadata Key", "Value"})

	hasDetails := false
	for _, p := range plugins {
		name := p["name"].(string)
		if metadata, ok := p["metadata"].(map[string]interface{}); ok && metadata != nil {
			for key, value := range metadata {
				valueStr := fmt.Sprintf("%v", value)
				if len(valueStr) > 50 {
					valueStr = valueStr[:47] + "..."
				}
				detailTable.Append([]string{name, key, valueStr})
				hasDetails = true
			}
		}
	}

	if hasDetails {
		detailTable.SetAutoMergeCells(true)
		detailTable.SetBorder(false)
		detailTable.SetCenterSeparator("|")
		detailTable.SetColumnSeparator("|")
		detailTable.SetRowSeparator("-")
		detailTable.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
		detailTable.SetAlignment(tablewriter.ALIGN_LEFT)
		fmt.Println("\n📋 Plugin Details:")
		detailTable.Render()
	}
}