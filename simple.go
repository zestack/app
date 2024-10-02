package app

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"zestack.dev/env"
	"zestack.dev/slim"
)

type simpleApp struct {
	config   Config
	servlets servlets
	slim     *slim.Slim
	started  chan struct{}
	exit     chan chan error
}

func New(config Config) App {
	return &simpleApp{
		config:  config,
		started: make(chan struct{}),
		exit:    make(chan chan error),
	}
}

type simpleServlet struct {
	Servlet
	init *servletInitContext
}

func (s *simpleApp) Use(servlets ...Servlet) error {
	select {
	case <-s.started:
		return errors.New("app: server already started")
	default:
	}
	if len(servlets) == 0 {
		return nil
	}
	for i, servlet := range servlets {
		for _, used := range s.servlets {
			if used.Name() == servlet.Name() {
				return fmt.Errorf(`app: servlet "%s" already registered`, servlet.Name())
			}
		}
		servlets[i] = &simpleServlet{
			Servlet: servlet,
			init:    nil,
		}
	}
	s.servlets = append(s.servlets, servlets...)
	return nil
}

func (s *simpleApp) Start() error {
	select {
	case <-s.started:
		return errors.New("app: server already started")
	default:
		close(s.started)
	}
	var err error
	call := func(fn func() error) {
		if err == nil {
			err = fn()
		}
	}
	call(s.ensureConfig)    // 确认必要的配置
	call(s.sortServlets)    // 按优先级排序
	call(s.initServlets)    // 初始化网络组件
	call(s.configureKernel) // 配置应用
	call(s.configureRoutes) // 注册路由
	call(s.bootServlets)    // 启动网络组件
	call(s.bootstrap)       // 启动网络服务器
	return err
}

func (s *simpleApp) ensureConfig() (err error) {
	return s.config.ensure()
}

func (s *simpleApp) sortServlets() error {
	if s.servlets.Len() == 0 {
		return errors.New("no registers servlet")
	}
	sort.Sort(s.servlets)
	return nil
}

func (s *simpleApp) initServlets() error {
	sort.Sort(s.servlets)
	for _, servlet := range s.servlets {
		ctx := NewServletInitContext().(*servletInitContext)
		servlet.(*simpleServlet).init = ctx
		err := servlet.Init(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *simpleApp) configureKernel() error {
	kernel := slim.New()
	kernel.Debug = !env.IsEnv("prod")
	kernel.Logger = s.config.Logger
	kernel.Use(slim.LoggingWithConfig(s.config.Logging))
	kernel.Use(slim.RecoveryWithConfig(s.config.Recover))
	kernel.Use(cors(s.config.CORS))
	s.config.Server.use(kernel)
	s.config.Routing.use(kernel)
	s.slim = kernel
	return nil
}

func (s *simpleApp) configureRoutes() error {
	// 按 Servlet 的顺序注册路由
	for _, servlet := range s.servlets {
		ctx := servlet.(*simpleServlet).init
		for _, route := range ctx.routes {
			route(s.slim)
		}
	}
	return nil
}

func (s *simpleApp) bootServlets() error {
	for _, servlet := range s.servlets {
		err := servlet.Bootstrap()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *simpleApp) bootstrap() error {
	go func() {
		srv := newServer(s)
		if srvErr := s.slim.StartServer(srv); srvErr != nil {
			slog.Error("encountered an error while serving listener: " + srvErr.Error())
		}
	}()
	// 监听停止命令，停止网络服务
	go func() {
		errChan := <-s.exit
		// 销毁注册的服务
		for i := len(s.servlets) - 1; i >= 0; i-- {
			if ex := s.servlets[i].Destroy(); ex != nil {
				slog.Warn("servlet Stop returned with error: " + ex.Error())
			}
		}
		// stop the listener
		errChan <- s.slim.Close()
	}()
	return nil
}

func (s *simpleApp) Stop() error {
	select {
	case <-s.started:
		exit := make(chan error)
		s.exit <- exit
		return <-exit
	default:
		return nil
	}
}
