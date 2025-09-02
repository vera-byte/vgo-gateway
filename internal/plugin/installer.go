package plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// PluginInstaller 插件安装器
// 支持从网络下载和安装插件
type PluginInstaller struct {
	// vpksDir vpks目录路径
	vpksDir string
	
	// logger 日志记录器
	logger *zap.Logger
	
	// httpClient HTTP客户端
	httpClient *http.Client
}

// PluginInfo 插件信息
type PluginInfo struct {
	// ServiceName 服务名称
	ServiceName string
	
	// Platform 平台架构
	Platform string
	
	// Version 版本号
	Version string
	
	// Filename 文件名
	Filename string
}

// NewPluginInstaller 创建新的插件安装器
// vpksDir: vpks目录路径
// logger: 日志记录器
// 返回: 插件安装器实例
func NewPluginInstaller(vpksDir string, logger *zap.Logger) *PluginInstaller {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	
	// 创建vpks目录
	if err := os.MkdirAll(vpksDir, 0755); err != nil {
		logger.Error("创建vpks目录失败", zap.String("dir", vpksDir), zap.Error(err))
	}
	
	return &PluginInstaller{
		vpksDir: vpksDir,
		logger:  logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// InstallFromURL 从URL安装插件
// pluginURL: 插件下载URL
// 返回: 本地文件路径和错误信息
func (i *PluginInstaller) InstallFromURL(ctx context.Context, pluginURL string) (string, error) {
	i.logger.Info("开始从URL安装插件", zap.String("url", pluginURL))
	
	// 验证URL格式
	if err := i.validateURL(pluginURL); err != nil {
		return "", fmt.Errorf("无效的URL: %w", err)
	}
	
	// 解析文件名
	filename, err := i.extractFilename(pluginURL)
	if err != nil {
		return "", fmt.Errorf("无法解析文件名: %w", err)
	}
	
	// 验证插件文件名格式
	if err := i.validatePluginFilename(filename); err != nil {
		return "", fmt.Errorf("无效的插件文件名: %w", err)
	}
	
	// 解析插件信息
	newPluginInfo, err := i.parsePluginInfo(filename)
	if err != nil {
		i.logger.Warn("无法解析插件信息，继续安装", 
			zap.String("filename", filename),
			zap.Error(err))
	} else {
		// 检查是否存在同一服务的其他版本
		existingPlugins, err := i.findPluginsByService(newPluginInfo.ServiceName)
		if err != nil {
			i.logger.Warn("查找已安装插件失败", zap.Error(err))
		} else if len(existingPlugins) > 0 {
			// 自动卸载同一服务的旧版本
			for _, existingPlugin := range existingPlugins {
				if existingPlugin.Filename != filename {
					i.logger.Info("检测到同一服务的旧版本，自动卸载", 
						zap.String("service", newPluginInfo.ServiceName),
						zap.String("old_version", existingPlugin.Version),
						zap.String("new_version", newPluginInfo.Version),
						zap.String("old_filename", existingPlugin.Filename))
					
					if err := i.RemovePlugin(existingPlugin.Filename); err != nil {
						i.logger.Error("卸载旧版本插件失败", 
							zap.String("filename", existingPlugin.Filename),
							zap.Error(err))
						return "", fmt.Errorf("卸载旧版本插件失败: %w", err)
					}
					
					i.logger.Info("旧版本插件卸载成功", 
						zap.String("filename", existingPlugin.Filename))
				}
			}
		}
	}
	
	// 构建本地文件路径
	localPath := filepath.Join(i.vpksDir, filename)
	
	// 检查文件是否已存在
	if _, err := os.Stat(localPath); err == nil {
		i.logger.Warn("插件文件已存在，将覆盖", zap.String("path", localPath))
	}
	
	// 下载文件
	if err := i.downloadFile(ctx, pluginURL, localPath); err != nil {
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	
	i.logger.Info("插件安装成功", 
		zap.String("url", pluginURL),
		zap.String("local_path", localPath))
	
	return localPath, nil
}

// validateURL 验证URL格式
// pluginURL: 插件URL
// 返回: 错误信息
func (i *PluginInstaller) validateURL(pluginURL string) error {
	parsedURL, err := url.Parse(pluginURL)
	if err != nil {
		return err
	}
	
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("不支持的协议: %s", parsedURL.Scheme)
	}
	
	if parsedURL.Host == "" {
		return fmt.Errorf("无效的主机名")
	}
	
	return nil
}

// extractFilename 从URL中提取文件名
// pluginURL: 插件URL
// 返回: 文件名和错误信息
func (i *PluginInstaller) extractFilename(pluginURL string) (string, error) {
	parsedURL, err := url.Parse(pluginURL)
	if err != nil {
		return "", err
	}
	
	filename := filepath.Base(parsedURL.Path)
	if filename == "." || filename == "/" {
		return "", fmt.Errorf("无法从URL中提取文件名")
	}
	
	return filename, nil
}

// validatePluginFilename 验证插件文件名格式
// filename: 文件名
// 返回: 错误信息
func (i *PluginInstaller) validatePluginFilename(filename string) error {
	// 检查文件扩展名
	if !strings.HasSuffix(filename, ".vkp") {
		return fmt.Errorf("插件文件必须以.vkp结尾")
	}
	
	// 验证命名格式: <服务名称>_<平台架构>_<版本>.vkp
	// 例如: example_linux_amd64_v1.0.0.vkp
	pattern := `^[a-zA-Z0-9-]+_[a-zA-Z0-9_]+_v?[0-9]+\.[0-9]+\.[0-9]+.*\.vkp$`
	matched, err := regexp.MatchString(pattern, filename)
	if err != nil {
		return fmt.Errorf("正则表达式错误: %w", err)
	}
	
	if !matched {
		i.logger.Warn("插件文件名不符合推荐格式", 
			zap.String("filename", filename),
			zap.String("expected_format", "<服务名称>_<平台架构>_<版本>.vkp"))
		// 不阻止安装，只是警告
	}
	
	return nil
}

// parsePluginInfo 解析插件信息
// filename: 插件文件名
// 返回: 插件信息和错误
func (i *PluginInstaller) parsePluginInfo(filename string) (*PluginInfo, error) {
	// 移除.vkp扩展名
	name := strings.TrimSuffix(filename, ".vkp")
	
	// 按下划线分割
	parts := strings.Split(name, "_")
	if len(parts) < 3 {
		return nil, fmt.Errorf("插件文件名格式不正确，应为: <服务名称>_<平台架构>_<版本>.vkp")
	}
	
	// 查找版本部分（以v开头或纯数字开头）
	versionIndex := -1
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// 检查是否为版本格式
		if matched, _ := regexp.MatchString(`^v?[0-9]+\.[0-9]+\.[0-9]+`, part); matched {
			versionIndex = i
			break
		}
	}
	
	if versionIndex == -1 {
		return nil, fmt.Errorf("无法识别版本信息")
	}
	
	// 服务名称是第一部分
	serviceName := parts[0]
	
	// 平台架构是版本之前的部分（可能有多个下划线）
	platform := strings.Join(parts[1:versionIndex], "_")
	
	// 版本是从版本索引开始的所有部分
	version := strings.Join(parts[versionIndex:], "_")
	
	return &PluginInfo{
		ServiceName: serviceName,
		Platform:    platform,
		Version:     version,
		Filename:    filename,
	}, nil
}

// downloadFile 下载文件
// url: 下载URL
// filepath: 本地文件路径
// 返回: 错误信息
func (i *PluginInstaller) downloadFile(ctx context.Context, url, filepath string) error {
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	// 设置User-Agent
	req.Header.Set("User-Agent", "vgo-gateway-plugin-installer/1.0")
	
	// 发送请求
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP错误: %d %s", resp.StatusCode, resp.Status)
	}
	
	// 创建本地文件
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	
	// 复制数据
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	
	i.logger.Info("文件下载完成", 
		zap.String("url", url),
		zap.String("filepath", filepath),
		zap.Int64("size", written))
	
	return nil
}

// ListInstalledPlugins 列出已安装的插件
// 返回: 插件文件列表和错误信息
func (i *PluginInstaller) ListInstalledPlugins() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(i.vpksDir, "*.vkp"))
	if err != nil {
		return nil, err
	}
	
	// 转换为相对路径
	var plugins []string
	for _, file := range files {
		plugins = append(plugins, filepath.Base(file))
	}
	
	return plugins, nil
}

// findPluginsByService 查找指定服务的已安装插件
// serviceName: 服务名称
// 返回: 插件信息列表和错误
func (i *PluginInstaller) findPluginsByService(serviceName string) ([]*PluginInfo, error) {
	plugins, err := i.ListInstalledPlugins()
	if err != nil {
		return nil, err
	}
	
	var servicePlugins []*PluginInfo
	for _, filename := range plugins {
		pluginInfo, err := i.parsePluginInfo(filename)
		if err != nil {
			i.logger.Warn("解析插件信息失败", 
				zap.String("filename", filename),
				zap.Error(err))
			continue
		}
		
		if pluginInfo.ServiceName == serviceName {
			servicePlugins = append(servicePlugins, pluginInfo)
		}
	}
	
	return servicePlugins, nil
}

// RemovePlugin 移除已安装的插件
// filename: 插件文件名
// 返回: 错误信息
func (i *PluginInstaller) RemovePlugin(filename string) error {
	filepath := filepath.Join(i.vpksDir, filename)
	
	// 检查文件是否存在
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return fmt.Errorf("插件文件不存在: %s", filename)
	}
	
	// 删除文件
	if err := os.Remove(filepath); err != nil {
		return fmt.Errorf("删除插件文件失败: %w", err)
	}
	
	i.logger.Info("插件已移除", zap.String("filename", filename))
	return nil
}