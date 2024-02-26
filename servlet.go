package app

import "zestack.dev/slim"

// Servlet 服务组件
type Servlet interface {
	// Name 返回服务组件名称
	Name() string
	// Priority 返回服务组件优先级
	Priority() int
	// Init 初始化服务组件
	Init(c ServletInitContext) error
	// Bootstrap 启动服务组件
	Bootstrap() error
	// Destroy 销毁服务组件
	Destroy() error
}

type servlets []Servlet

func (s servlets) Len() int           { return len(s) }
func (s servlets) Less(i, j int) bool { return s[i].Priority() < s[j].Priority() }
func (s servlets) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type Controller interface {
	InitRoutes(r slim.RouteCollector)
}

type ControllerFunc func(r slim.RouteCollector)

func (f ControllerFunc) InitRoutes(r slim.RouteCollector) {
	f(r)
}

type ServletInitContext interface {
	// Use 注册中间件
	Use(middleware ...slim.MiddlewareFunc)
	// Routes 注册路由，会共享 Use 方法注册的中间件
	Routes(controllers ...Controller)
	// Hosts 注册与指定 host 相关的路由，会共享 Use 方法注册的中间件
	Hosts(host string, controllers ...Controller)
}

func NewServletInitContext() ServletInitContext {
	return &servletInitContext{}
}

type servletInitContext struct {
	middleware []slim.MiddlewareFunc
	routes     []func(*slim.Slim)
}

// Use 注册中间件
func (c *servletInitContext) Use(middleware ...slim.MiddlewareFunc) {
	c.middleware = append(c.middleware, middleware...)
}

// Routes 注册路由
func (c *servletInitContext) Routes(controllers ...Controller) {
	c.routes = append(c.routes, func(s *slim.Slim) {
		s.Router().Group(func(r slim.RouteCollector) {
			r.Use(c.middleware...)
			for _, controller := range controllers {
				controller.InitRoutes(r)
			}
		})
	})
}

// Hosts 注册与指定 host 相关的路由
func (c *servletInitContext) Hosts(host string, controllers ...Controller) {
	c.routes = append(c.routes, func(s *slim.Slim) {
		router := s.RouterFor(host)
		if router == nil {
			router = s.Host(host)
		}
		router.Group(func(r slim.RouteCollector) {
			r.Use(c.middleware...)
			for _, controller := range controllers {
				controller.InitRoutes(r)
			}
		})
	})
}
