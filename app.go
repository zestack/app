package app

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

type App interface {
	Use(servlets ...Servlet) error
	Start() error
	Stop() error
}

// 全局应用单例
var app = New(Config{}).(*simpleApp)

func Init(config Config) error {
	app.config = config
	return nil
}

// Use 注册服务组件
func Use(servlets ...Servlet) error {
	return app.Use(servlets...)
}

// Start 启动应用程序
// 使用该方法前，必须先调用 Init 方法
func Start() error {
	return app.Start()
}

// Stop 停止应用
func Stop() error {
	return app.Stop()
}

// Run 运行应用
// 使用该方法前，必须先调用 Init 方法
func Run() error {
	if err := Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	slog.Info("Received signal " + (<-ch).String())

	return Stop()
}
