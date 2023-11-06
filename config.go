package app

import "zestack.dev/slim"

type Config struct {
	// Server 服务器配置
	Server ServerConfig
	// Logging 日志配置
	Logging LoggingConfig
	// CORS 跨域配置
	CORS CORSConfig
	// Recover 错误拦截配置
	Recover slim.RecoverConfig
}
