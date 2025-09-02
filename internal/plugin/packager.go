package plugin

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// VKPPackager VKP打包器
// 负责将模块编译和打包为.vkp文件
type VKPPackager struct {
	// logger 日志记录器
	logger *zap.Logger
	
	// workDir 工作目录
	workDir string
}

// PackageConfig 打包配置
type PackageConfig struct {
	// ModulePath 模块路径
	ModulePath string `json:"module_path"`
	
	// OutputPath 输出路径
	OutputPath string `json:"output_path"`
	
	// Metadata 插件元数据
	Metadata *PluginMetadata `json:"metadata"`
	
	// BuildTags 构建标签
	BuildTags []string `json:"build_tags,omitempty"`
	
	// GOOS 目标操作系统
	GOOS string `json:"goos,omitempty"`
	
	// GOARCH 目标架构
	GOARCH string `json:"goarch,omitempty"`
	
	// LDFlags 链接器标志
	LDFlags string `json:"ldflags,omitempty"`
	
	// IncludeFiles 包含的额外文件
	IncludeFiles []string `json:"include_files,omitempty"`
}

// NewVKPPackager 创建新的VKP打包器
// logger: 日志记录器
// workDir: 工作目录
// 返回: VKP打包器实例
func NewVKPPackager(logger *zap.Logger, workDir string) *VKPPackager {
	return &VKPPackager{
		logger:  logger,
		workDir: workDir,
	}
}

// Package 打包模块为VKP文件
// config: 打包配置
// 返回: 错误信息
func (p *VKPPackager) Package(config *PackageConfig) error {
	p.logger.Info("Starting VKP packaging", 
		zap.String("module", config.ModulePath),
		zap.String("output", config.OutputPath))
	
	// 创建临时目录
	tempDir, err := os.MkdirTemp(p.workDir, "vkp-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// 编译模块
	binaryPath, err := p.buildModule(config, tempDir)
	if err != nil {
		return fmt.Errorf("failed to build module: %w", err)
	}
	
	// 创建VKP包
	if err := p.createVKPPackage(config, binaryPath, tempDir); err != nil {
		return fmt.Errorf("failed to create VKP package: %w", err)
	}
	
	p.logger.Info("VKP packaging completed", zap.String("output", config.OutputPath))
	return nil
}

// buildModule 编译模块
// config: 打包配置
// tempDir: 临时目录
// 返回: 二进制文件路径和错误信息
func (p *VKPPackager) buildModule(config *PackageConfig, tempDir string) (string, error) {
	binaryName := filepath.Base(config.ModulePath)
	if config.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)
	
	// 构建go build命令
	args := []string{"build"}
	
	// 添加构建标签
	if len(config.BuildTags) > 0 {
		args = append(args, "-tags", strings.Join(config.BuildTags, ","))
	}
	
	// 添加链接器标志
	if config.LDFlags != "" {
		args = append(args, "-ldflags", config.LDFlags)
	}
	
	// 添加输出路径
	args = append(args, "-o", binaryPath)
	
	// 添加模块路径
	args = append(args, config.ModulePath)
	
	// 创建命令
	cmd := exec.Command("go", args...)
	
	// 设置环境变量
	env := os.Environ()
	if config.GOOS != "" {
		env = append(env, "GOOS="+config.GOOS)
	}
	if config.GOARCH != "" {
		env = append(env, "GOARCH="+config.GOARCH)
	}
	cmd.Env = env
	
	// 设置工作目录
	cmd.Dir = filepath.Dir(config.ModulePath)
	
	p.logger.Info("Building module", 
		zap.String("command", cmd.String()),
		zap.String("output", binaryPath))
	
	// 执行编译
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\nOutput: %s", err, string(output))
	}
	
	return binaryPath, nil
}

// createVKPPackage 创建VKP包
// config: 打包配置
// binaryPath: 二进制文件路径
// tempDir: 临时目录
// 返回: 错误信息
func (p *VKPPackager) createVKPPackage(config *PackageConfig, binaryPath, tempDir string) error {
	// 创建元数据文件
	metadataPath := filepath.Join(tempDir, "metadata.json")
	if err := p.createMetadataFile(config.Metadata, metadataPath); err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	
	// 创建VKP文件
	vkpFile, err := os.Create(config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create VKP file: %w", err)
	}
	defer vkpFile.Close()
	
	// 创建gzip压缩器
	gzWriter := gzip.NewWriter(vkpFile)
	defer gzWriter.Close()
	
	// 创建tar归档器
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()
	
	// 添加二进制文件
	if err := p.addFileToTar(tarWriter, binaryPath, "plugin"); err != nil {
		return fmt.Errorf("failed to add binary to VKP: %w", err)
	}
	
	// 添加元数据文件
	if err := p.addFileToTar(tarWriter, metadataPath, "metadata.json"); err != nil {
		return fmt.Errorf("failed to add metadata to VKP: %w", err)
	}
	
	// 添加额外文件
	for _, filePath := range config.IncludeFiles {
		if _, err := os.Stat(filePath); err == nil {
			fileName := filepath.Base(filePath)
			if err := p.addFileToTar(tarWriter, filePath, fileName); err != nil {
				p.logger.Warn("Failed to add file to VKP", 
					zap.String("file", filePath),
					zap.Error(err))
			}
		}
	}
	
	return nil
}

// createMetadataFile 创建元数据文件
// metadata: 插件元数据
// filePath: 文件路径
// 返回: 错误信息
func (p *VKPPackager) createMetadataFile(metadata *PluginMetadata, filePath string) error {
	if metadata == nil {
		metadata = &PluginMetadata{
			Name:        "unknown",
			Version:     "1.0.0",
			Description: "VKP Plugin",
			Standalone:  true,
		}
	}
	
	// 添加构建时间
	if metadata.APIVersion == "" {
		metadata.APIVersion = "v1"
	}
	
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	return os.WriteFile(filePath, data, 0644)
}

// addFileToTar 添加文件到tar归档
// tarWriter: tar写入器
// filePath: 文件路径
// tarPath: tar中的路径
// 返回: 错误信息
func (p *VKPPackager) addFileToTar(tarWriter *tar.Writer, filePath, tarPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()
	
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	
	header := &tar.Header{
		Name:    tarPath,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}
	
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	
	if _, err := io.Copy(tarWriter, file); err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}
	
	return nil
}

// PackageFromConfig 从配置文件打包
// configPath: 配置文件路径
// 返回: 错误信息
func (p *VKPPackager) PackageFromConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config PackageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return p.Package(&config)
}

// GeneratePackageConfig 生成打包配置模板
// modulePath: 模块路径
// outputPath: 输出路径
// 返回: 打包配置和错误信息
func GeneratePackageConfig(modulePath, outputPath string) (*PackageConfig, error) {
	moduleName := filepath.Base(modulePath)
	
	config := &PackageConfig{
		ModulePath: modulePath,
		OutputPath: outputPath,
		Metadata: &PluginMetadata{
			Name:               moduleName,
			Version:            "1.0.0",
			Description:        fmt.Sprintf("%s plugin", moduleName),
			Author:             "Unknown",
			License:            "MIT",
			APIVersion:         "v1",
			MinGatewayVersion:  "1.0.0",
			Standalone:         true,
			Dependencies:       []string{},
		},
		GOOS:   "linux",
		GOARCH: "amd64",
	}
	
	return config, nil
}