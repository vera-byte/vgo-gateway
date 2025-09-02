package main

import (
	cmd "github.com/vera-byte/vgo-gateway/cmd"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

// main VGO Gateway服务器主入口
// 启动VGO Gateway服务器
func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		vgokit.Log.Fatal("Failed to execute command", zap.Error(err))
	}
}
