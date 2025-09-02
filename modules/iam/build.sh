#!/bin/bash

# IAM模块构建脚本
# 支持独立运行、测试和VKP打包

set -e

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$SCRIPT_DIR"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    echo "IAM Module Build Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  build       构建模块二进制文件"
    echo "  test        运行测试"
    echo "  run         独立运行模块"
    echo "  package     打包为VKP文件"
    echo "  clean       清理构建文件"
    echo "  help        显示帮助信息"
    echo ""
    echo "Options:"
    echo "  --port PORT    指定运行端口 (默认: 8080)"
    echo "  --os OS        目标操作系统 (默认: linux)"
    echo "  --arch ARCH    目标架构 (默认: amd64)"
    echo "  --output PATH  输出路径"
    echo ""
    echo "Examples:"
    echo "  $0 build"
    echo "  $0 test"
    echo "  $0 run --port 8081"
    echo "  $0 package --os linux --arch amd64"
}

# 构建模块
build_module() {
    local os=${1:-linux}
    local arch=${2:-amd64}
    local output=${3:-"./bin/iam-module"}
    
    log_info "构建IAM模块..."
    log_info "目标平台: $os/$arch"
    log_info "输出路径: $output"
    
    # 创建输出目录
    mkdir -p "$(dirname "$output")"
    
    # 设置构建变量
    local version="1.0.0"
    local build_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    local ldflags="-s -w -X main.version=$version -X main.buildTime=$build_time"
    
    # 构建
    cd "$MODULE_DIR"
    GOOS="$os" GOARCH="$arch" go build \
        -tags plugin \
        -ldflags "$ldflags" \
        -o "$output" \
        ./cmd
    
    if [ $? -eq 0 ]; then
        log_success "构建完成: $output"
        
        # 显示文件信息
        if [ -f "$output" ]; then
            local size=$(du -h "$output" | cut -f1)
            log_info "文件大小: $size"
        fi
    else
        log_error "构建失败"
        exit 1
    fi
}

# 运行测试
run_tests() {
    log_info "运行IAM模块测试..."
    
    cd "$MODULE_DIR"
    
    # 运行单元测试
    go test -v ./...
    
    if [ $? -eq 0 ]; then
        log_success "所有测试通过"
    else
        log_error "测试失败"
        exit 1
    fi
}

# 独立运行模块
run_module() {
    local port=${1:-8080}
    
    log_info "独立运行IAM模块..."
    log_info "监听端口: $port"
    
    # 先构建模块
    build_module "$(go env GOOS)" "$(go env GOARCH)" "./bin/iam-module"
    
    # 运行模块
    cd "$MODULE_DIR"
    ./bin/iam-module -mode standalone -port "$port"
}

# 打包为VKP文件
package_vkp() {
    local os=${1:-linux}
    local arch=${2:-amd64}
    local output=${3:-"./iam.vkp"}
    
    log_info "打包IAM模块为VKP文件..."
    log_info "目标平台: $os/$arch"
    log_info "输出路径: $output"
    
    # 检查配置文件
    if [ ! -f "$MODULE_DIR/vkp-config.json" ]; then
        log_error "VKP配置文件不存在: vkp-config.json"
        exit 1
    fi
    
    # 创建临时目录
    local temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT
    
    # 构建二进制文件
    local binary_path="$temp_dir/iam-module"
    if [ "$os" = "windows" ]; then
        binary_path="$binary_path.exe"
    fi
    
    build_module "$os" "$arch" "$binary_path"
    
    # 创建元数据文件
    local metadata_file="$temp_dir/metadata.json"
    jq '.metadata' "$MODULE_DIR/vkp-config.json" > "$metadata_file"
    
    # 创建插件元数据文件
    cat > "$temp_dir/plugin.json" << EOF
{
    "name": "iam",
    "version": "1.0.0",
    "description": "Identity and Access Management module",
    "author": "VGO Team",
    "license": "MIT",
    "api_version": "v1",
    "min_gateway_version": "1.0.0",
    "standalone_support": true,
    "binary_name": "$(basename $binary_path)",
    "config_schema": {
        "endpoint": {
            "type": "string",
            "default": "localhost:9090",
            "description": "IAM service endpoint"
        },
        "timeout": {
            "type": "integer",
            "default": 30,
            "description": "Connection timeout in seconds"
        }
    }
}
EOF
    
    # 创建VKP包
     cd "$temp_dir"
     
     # 重命名二进制文件为plugin
     mv "$(basename "$binary_path")" plugin
     
     tar -czf "$MODULE_DIR/$output" plugin metadata.json plugin.json
    
    if [ $? -eq 0 ]; then
        log_success "VKP打包完成: $output"
        
        # 显示文件信息
        if [ -f "$MODULE_DIR/$output" ]; then
            local size=$(du -h "$MODULE_DIR/$output" | cut -f1)
            log_info "VKP文件大小: $size"
            log_info "VKP内容:"
            tar -tzf "$MODULE_DIR/$output" | sed 's/^/  - /'
        fi
    else
        log_error "VKP打包失败"
        exit 1
    fi
}

# 清理构建文件
clean_build() {
    log_info "清理构建文件..."
    
    cd "$MODULE_DIR"
    
    # 删除构建产物
    rm -rf bin/
    rm -f *.vkp
    rm -f *.so
    
    # 清理Go缓存
    go clean -cache
    go clean -modcache
    
    log_success "清理完成"
}

# 解析命令行参数
COMMAND=""
PORT=8080
OS="linux"
ARCH="amd64"
OUTPUT=""

while [[ $# -gt 0 ]]; do
    case $1 in
        build|test|run|package|clean|help)
            COMMAND="$1"
            shift
            ;;
        --port)
            PORT="$2"
            shift 2
            ;;
        --os)
            OS="$2"
            shift 2
            ;;
        --arch)
            ARCH="$2"
            shift 2
            ;;
        --output)
            OUTPUT="$2"
            shift 2
            ;;
        *)
            log_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 执行命令
case "$COMMAND" in
    build)
        if [ -n "$OUTPUT" ]; then
            build_module "$OS" "$ARCH" "$OUTPUT"
        else
            build_module "$OS" "$ARCH"
        fi
        ;;
    test)
        run_tests
        ;;
    run)
        run_module "$PORT"
        ;;
    package)
        if [ -n "$OUTPUT" ]; then
            package_vkp "$OS" "$ARCH" "$OUTPUT"
        else
            package_vkp "$OS" "$ARCH"
        fi
        ;;
    clean)
        clean_build
        ;;
    help|"")
        show_help
        ;;
    *)
        log_error "未知命令: $COMMAND"
        show_help
        exit 1
        ;;
esac