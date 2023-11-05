package app

import (
	"os"
	"os/signal"
	"syscall"

	"zestack.dev/log"
)

type App interface {
	Init(options ...Option) error
	Use(servlets ...Servlet) error
	Start() error
	Stop() error
}

var app App

func Init(options ...Option) error {
	if app != nil {
		return app.Init(options...)
	}
	app = New(options...)
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
	if app == nil {
		return nil
	}
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
	log.Info("Received signal %s", <-ch)

	return Stop()
}
