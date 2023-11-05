package app

import "zestack.dev/slim"

type ServletConfig interface {
	// SetRoutes 设置路由
	SetRoutes(prefix string, fn func(r slim.RouteCollector))
	// SetHostingRoutes 配置指定主机的路由
	SetHostingRoutes(host, prefix string, fn func(r slim.RouteCollector))
	// Use 注册中间件
	Use(middleware ...slim.MiddlewareFunc)
}

// Servlet 服务组件
type Servlet interface {
	// Name 返回服务组件名称
	Name() string
	// Priority 返回服务组件优先级
	Priority() int
	// Init 初始化服务组件
	Init(ServletConfig) error
	// Bootstrap 启动服务组件
	Bootstrap() error
	// Destroy 销毁服务组件
	Destroy() error
}

type servlets []Servlet

func (s servlets) Len() int           { return len(s) }
func (s servlets) Less(i, j int) bool { return s[i].Priority() > s[j].Priority() }
func (s servlets) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type servletConfig struct {
	routes     []routing
	middleware []slim.MiddlewareFunc
}

type routing struct {
	host     string
	prefix   string
	register func(slim.RouteCollector)
}

func (cfg *servletConfig) Use(middleware ...slim.MiddlewareFunc) {
	cfg.middleware = append(cfg.middleware, middleware...)
}

func (cfg *servletConfig) SetRoutes(prefix string, fn func(r slim.RouteCollector)) {
	cfg.SetHostingRoutes("", prefix, fn)
}

func (cfg *servletConfig) SetHostingRoutes(host, prefix string, fn func(r slim.RouteCollector)) {
	cfg.routes = append(cfg.routes, routing{
		host:   host,
		prefix: prefix,
		register: func(c slim.RouteCollector) {
			c.Use(cfg.middleware...)
			fn(c)
		},
	})
}
