package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VKPLoader VKP插件加载器
// 支持加载.vkp格式的插件文件
type VKPLoader struct {
	// loadedPlugins 已加载的插件映射
	loadedPlugins map[string]*LoadedPlugin
	
	// logger 日志记录器
	logger *zap.Logger
	
	// pluginDir 插件存储目录
	pluginDir string
	
	// mu 读写锁
	mu sync.RWMutex
}

// LoadedPlugin 已加载的插件信息
type LoadedPlugin struct {
	// Plugin 插件实例
	Plugin Plugin
	
	// Path 插件文件路径
	Path string
	
	// ExtractDir 解压目录
	ExtractDir string
	
	// Handle 插件句柄（用于Go plugin）
	Handle *plugin.Plugin
}

// NewVKPLoader 创建新的VKP插件加载器
// pluginDir: 插件存储目录
// logger: 日志记录器
// 返回: VKP插件加载器实例
func NewVKPLoader(pluginDir string, logger *zap.Logger) *VKPLoader {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	
	return &VKPLoader{
		loadedPlugins: make(map[string]*LoadedPlugin),
		logger:        logger,
		pluginDir:     pluginDir,
	}
}

// LoadPlugin 加载插件
// path: 插件文件路径
// 返回: 插件实例和错误信息
func (l *VKPLoader) LoadPlugin(path string) (Plugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file not found: %s", path)
	}
	
	// 根据文件扩展名选择加载方式
	ext := filepath.Ext(path)
	switch ext {
	case ".vkp":
		return l.loadVKPPlugin(path)
	case ".so":
		return l.loadGoPlugin(path)
	default:
		return nil, fmt.Errorf("unsupported plugin format: %s", ext)
	}
}

// loadVKPPlugin 加载VKP插件
// path: VKP文件路径
// 返回: 插件实例和错误信息
func (l *VKPLoader) loadVKPPlugin(path string) (Plugin, error) {
	l.logger.Info("开始加载VKP插件", zap.String("path", path))
	
	// 创建临时解压目录
	extractDir := filepath.Join(l.pluginDir, "temp", filepath.Base(path)+"_extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("创建解压目录失败: %w", err)
	}
	
	// 解压VKP文件
	if err := l.extractVKP(path, extractDir); err != nil {
		return nil, fmt.Errorf("解压VKP文件失败: %w", err)
	}
	
	// 读取插件元数据
	metadata, err := l.readPluginMetadata(extractDir)
	if err != nil {
		return nil, fmt.Errorf("读取插件元数据失败: %w", err)
	}
	
	// 设置插件二进制文件权限
	binaryPath := filepath.Join(extractDir, "plugin")
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return nil, fmt.Errorf("设置插件二进制文件权限失败: %w", err)
	}
	
	// 创建VKP插件包装器
	vkpPlugin := &VKPPlugin{
		execPath: binaryPath,
		metadata: metadata,
		logger:   l.logger,
	}
	
	name := vkpPlugin.GetName()
	l.loadedPlugins[name] = &LoadedPlugin{
		Plugin:     vkpPlugin,
		Path:       path,
		ExtractDir: extractDir,
	}
	
	l.logger.Info("VKP插件加载成功", 
		zap.String("name", name),
		zap.String("version", metadata.Version),
		zap.String("path", path))
	
	return vkpPlugin, nil
}

// loadGoPlugin 加载Go插件
// path: .so文件路径
// 返回: 插件实例和错误信息
func (l *VKPLoader) loadGoPlugin(path string) (Plugin, error) {
	// 打开Go插件
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open go plugin: %w", err)
	}
	
	// 查找插件工厂函数
	sym, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export NewPlugin function: %w", err)
	}
	
	// 类型断言
	newPluginFunc, ok := sym.(func() Plugin)
	if !ok {
		return nil, fmt.Errorf("NewPlugin function has wrong signature")
	}
	
	// 创建插件实例
	pluginInstance := newPluginFunc()
	name := pluginInstance.GetName()
	
	l.loadedPlugins[name] = &LoadedPlugin{
		Plugin: pluginInstance,
		Path:   path,
		Handle: p,
	}
	
	l.logger.Info("Go plugin loaded", 
		zap.String("name", name),
		zap.String("path", path))
	
	return pluginInstance, nil
}

// extractVKP 解压VKP文件
// vkpPath: VKP文件路径
// extractDir: 解压目录
// 返回: 错误信息
func (l *VKPLoader) extractVKP(vkpPath, extractDir string) error {
	file, err := os.Open(vkpPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()
	
	tr := tar.NewReader(gzr)
	
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		target := filepath.Join(extractDir, header.Name)
		
		switch header.Typeflag {
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	
	return nil
}

// readPluginMetadata 读取插件元数据
// extractDir: 解压目录
// 返回: 插件元数据和错误信息
func (l *VKPLoader) readPluginMetadata(extractDir string) (*PluginMetadata, error) {
	metadataPath := filepath.Join(extractDir, "plugin.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}
	
	var metadata PluginMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	
	return &metadata, nil
}

// UnloadPlugin 卸载插件
// name: 插件名称
// 返回: 错误信息
func (l *VKPLoader) UnloadPlugin(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	loadedPlugin, exists := l.loadedPlugins[name]
	if !exists {
		return fmt.Errorf("plugin '%s' not loaded", name)
	}
	
	// 如果是VKP插件，需要停止进程
	if vkpPlugin, ok := loadedPlugin.Plugin.(*VKPPlugin); ok {
		if err := vkpPlugin.stop(); err != nil {
			l.logger.Warn("停止VKP插件失败", 
				zap.String("name", name),
				zap.Error(err))
		}
	}
	
	// 清理解压目录
	if loadedPlugin.ExtractDir != "" {
		if err := os.RemoveAll(loadedPlugin.ExtractDir); err != nil {
			l.logger.Warn("清理插件解压目录失败", 
				zap.String("plugin", name),
				zap.String("dir", loadedPlugin.ExtractDir),
				zap.Error(err))
		}
	}
	
	delete(l.loadedPlugins, name)
	l.logger.Info("插件卸载成功", zap.String("name", name))
	return nil
}

// ListPlugins 列出已加载的插件
// 返回: 插件名称列表
func (l *VKPLoader) ListPlugins() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	var names []string
	for name := range l.loadedPlugins {
		names = append(names, name)
	}
	return names
}

// GetPlugin 获取插件实例
// name: 插件名称
// 返回: 插件实例和是否存在
func (l *VKPLoader) GetPlugin(name string) (Plugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	loadedPlugin, exists := l.loadedPlugins[name]
	if !exists {
		return nil, false
	}
	return loadedPlugin.Plugin, true
}

// VKPPlugin VKP插件包装器
// 将外部可执行文件包装为插件接口
type VKPPlugin struct {
	// execPath 可执行文件路径
	execPath string
	
	// metadata 插件元数据
	metadata *PluginMetadata
	
	// logger 日志记录器
	logger *zap.Logger
	
	// cmd 运行中的命令
	cmd *exec.Cmd
}

// loadMetadata 加载插件元数据（已废弃，元数据现在通过plugin.json文件加载）
// 返回: 错误信息
func (p *VKPPlugin) loadMetadata() error {
	// 此方法已废弃，元数据现在通过plugin.json文件在加载时读取
	if p.metadata == nil {
		p.metadata = &PluginMetadata{
			Name:        filepath.Base(p.execPath),
			Version:     "1.0.0",
			Description: "VKP Plugin",
			Standalone:  true,
		}
	}
	return nil
}

// GetName 获取插件名称
// 返回: 插件名称
func (p *VKPPlugin) GetName() string {
	if p.metadata != nil {
		return p.metadata.Name
	}
	return filepath.Base(p.execPath)
}

// GetVersion 获取插件版本
// 返回: 插件版本
func (p *VKPPlugin) GetVersion() string {
	if p.metadata != nil {
		return p.metadata.Version
	}
	return "1.0.0"
}

// GetDescription 获取插件描述
// 返回: 插件描述
func (p *VKPPlugin) GetDescription() string {
	if p.metadata != nil {
		return p.metadata.Description
	}
	return "VKP Plugin"
}

// GetMetadata 获取插件元数据
// 返回: 插件元数据
func (p *VKPPlugin) GetMetadata() *PluginMetadata {
	return p.metadata
}

// CanRunStandalone 是否支持独立运行
// 返回: 是否支持独立运行
func (p *VKPPlugin) CanRunStandalone() bool {
	return true
}

// stop 停止VKP插件进程
// 返回: 错误信息
func (p *VKPPlugin) stop() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

// Initialize 初始化插件
// ctx: 上下文
// logger: 日志记录器
// config: 配置数据
// 返回: 错误信息
func (p *VKPPlugin) Initialize(ctx context.Context, logger *zap.Logger, config interface{}) error {
	// 通过命令行参数启动VKP插件
	cmd := exec.CommandContext(ctx, p.execPath, "--mode=gateway")
	p.cmd = cmd
	
	// 启动插件进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start VKP plugin: %w", err)
	}
	
	p.logger.Info("VKP plugin initialized", zap.String("name", p.GetName()))
	return nil
}

// RegisterRoutes 注册路由
// router: Gin路由器
// 返回: 错误信息
func (p *VKPPlugin) RegisterRoutes(router *gin.Engine) error {
	// VKP插件通过HTTP代理方式注册路由
	// 这里简化处理，实际需要通过IPC获取路由信息
	group := router.Group("/api/v1/" + p.GetName())
	group.Any("/*path", p.proxyHandler)
	
	p.logger.Info("VKP plugin routes registered", zap.String("name", p.GetName()))
	return nil
}

// Health 健康检查
// 返回: 健康状态和错误信息
func (p *VKPPlugin) Health() (map[string]interface{}, error) {
	// 检查VKP进程是否运行
	if p.cmd == nil || p.cmd.Process == nil {
		return map[string]interface{}{
			"status": "stopped",
		}, nil
	}
	
	// 简单的进程存活检查
	if err := p.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}, err
	}
	
	return map[string]interface{}{
		"status": "healthy",
		"pid":    p.cmd.Process.Pid,
	}, nil
}

// Shutdown 关闭插件
// ctx: 上下文
// 返回: 错误信息
func (p *VKPPlugin) Shutdown(ctx context.Context) error {
	return p.stop()
}

// RunStandalone 独立运行模式
// ctx: 上下文
// port: 监听端口
// 返回: 错误信息
func (p *VKPPlugin) RunStandalone(ctx context.Context, port int) error {
	cmd := exec.CommandContext(ctx, p.execPath, "--mode=standalone", fmt.Sprintf("--port=%d", port))
	return cmd.Run()
}

// proxyHandler 代理处理器
// 将请求转发到VKP插件进程
func (p *VKPPlugin) proxyHandler(c *gin.Context) {
	// 这里应该实现HTTP代理逻辑
	// 将请求转发到VKP插件的HTTP服务
	c.JSON(200, gin.H{
		"message": "VKP plugin proxy",
		"plugin":  p.GetName(),
		"path":    c.Request.URL.Path,
	})
}