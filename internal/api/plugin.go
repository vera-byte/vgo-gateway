package api

import (
	"context"
	"net/http"
	"time"

	"github.com/vera-byte/vgo-gateway/internal/plugin"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PluginHandler 插件API处理器
type PluginHandler struct {
	// pluginManager 插件管理器
	pluginManager *plugin.Manager
	
	// logger 日志记录器
	logger *zap.Logger
}

// NewPluginHandler 创建新的插件API处理器
// pluginManager: 插件管理器
// logger: 日志记录器
// 返回: 插件API处理器实例
func NewPluginHandler(pluginManager *plugin.Manager, logger *zap.Logger) *PluginHandler {
	return &PluginHandler{
		pluginManager: pluginManager,
		logger:        logger,
	}
}

// InstallPluginRequest 安装插件请求
type InstallPluginRequest struct {
	// URL 插件下载URL
	URL string `json:"url" binding:"required"`
	
	// AutoLoad 是否自动加载插件
	AutoLoad bool `json:"auto_load"`
}

// InstallPluginResponse 安装插件响应
type InstallPluginResponse struct {
	// Success 是否成功
	Success bool `json:"success"`
	
	// Message 消息
	Message string `json:"message"`
	
	// PluginName 插件名称（仅在自动加载时返回）
	PluginName string `json:"plugin_name,omitempty"`
}

// InstallPlugin 安装插件
// c: Gin上下文
func (h *PluginHandler) InstallPlugin(c *gin.Context) {
	var req InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, InstallPluginResponse{
			Success: false,
			Message: "无效的请求参数: " + err.Error(),
		})
		return
	}
	
	h.logger.Info("收到插件安装请求", 
		zap.String("url", req.URL),
		zap.Bool("auto_load", req.AutoLoad))
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	
	if req.AutoLoad {
		// 安装并加载插件
		if err := h.pluginManager.InstallAndLoadPluginFromURL(ctx, req.URL); err != nil {
			h.logger.Error("安装并加载插件失败", 
				zap.String("url", req.URL),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, InstallPluginResponse{
				Success: false,
				Message: "安装并加载插件失败: " + err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusOK, InstallPluginResponse{
			Success: true,
			Message: "插件安装并加载成功",
		})
	} else {
		// 仅安装插件
		if err := h.pluginManager.InstallPluginFromURL(ctx, req.URL); err != nil {
			h.logger.Error("安装插件失败", 
				zap.String("url", req.URL),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, InstallPluginResponse{
				Success: false,
				Message: "安装插件失败: " + err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusOK, InstallPluginResponse{
			Success: true,
			Message: "插件安装成功",
		})
	}
}

// ListInstalledPluginsResponse 列出已安装插件响应
type ListInstalledPluginsResponse struct {
	// Success 是否成功
	Success bool `json:"success"`
	
	// Message 消息
	Message string `json:"message"`
	
	// Plugins 插件列表
	Plugins []string `json:"plugins"`
}

// ListInstalledPlugins 列出已安装的插件
// c: Gin上下文
func (h *PluginHandler) ListInstalledPlugins(c *gin.Context) {
	plugins, err := h.pluginManager.ListInstalledPlugins()
	if err != nil {
		h.logger.Error("获取已安装插件列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ListInstalledPluginsResponse{
			Success: false,
			Message: "获取已安装插件列表失败: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, ListInstalledPluginsResponse{
		Success: true,
		Message: "获取已安装插件列表成功",
		Plugins: plugins,
	})
}

// RemovePluginRequest 移除插件请求
type RemovePluginRequest struct {
	// Filename 插件文件名
	Filename string `json:"filename" binding:"required"`
}

// RemovePluginResponse 移除插件响应
type RemovePluginResponse struct {
	// Success 是否成功
	Success bool `json:"success"`
	
	// Message 消息
	Message string `json:"message"`
}

// RemovePlugin 移除插件
// c: Gin上下文
func (h *PluginHandler) RemovePlugin(c *gin.Context) {
	var req RemovePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RemovePluginResponse{
			Success: false,
			Message: "无效的请求参数: " + err.Error(),
		})
		return
	}
	
	h.logger.Info("收到插件移除请求", zap.String("filename", req.Filename))
	
	if err := h.pluginManager.RemoveInstalledPlugin(req.Filename); err != nil {
		h.logger.Error("移除插件失败", 
			zap.String("filename", req.Filename),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, RemovePluginResponse{
			Success: false,
			Message: "移除插件失败: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, RemovePluginResponse{
		Success: true,
		Message: "插件移除成功",
	})
}

// RegisterRoutes 注册插件API路由
// router: Gin路由器
func (h *PluginHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/plugins")
	{
		// 安装插件
		api.POST("/install", h.InstallPlugin)
		
		// 列出已安装的插件
		api.GET("/installed", h.ListInstalledPlugins)
		
		// 移除插件
		api.DELETE("/remove", h.RemovePlugin)
	}
}