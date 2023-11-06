package app

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"

	"zestack.dev/app/middleware"
	"zestack.dev/env"
	"zestack.dev/log"
	"zestack.dev/slim"
)

type SimpleApp struct {
	options  *Options
	servlets servlets
	status   int32
	slim     *slim.Slim
	inits    map[string]*servletInitContext
	started  chan struct{}
	exit     chan chan error
}

func New(options ...Option) App {
	s := &SimpleApp{}
	_ = s.Init(options...)
	return s
}

func (s *SimpleApp) Init(options ...Option) error {
	select {
	case <-s.started:
		return errors.New("server already started")
	default:
	}
	s.options = &Options{
		maxHeaderBytes: env.Int("SERVER_MAX_HEADER_BYTES", http.DefaultMaxHeaderBytes),
		readTimeout:    env.Duration("SERVER_READ_TIMEOUT", 10*time.Second),
		writeTimeout:   env.Duration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		idleTimeout:    env.Duration("SERVER_IDLE_TIMEOUT", 120*time.Second),
	}
	for _, option := range options {
		option(s.options)
	}
	if s.slim != nil {
		s.slim = nil
	}
	if s.exit == nil {
		s.exit = make(chan chan error)
	}
	return nil
}

func (s *SimpleApp) Use(servlets ...Servlet) error {
	select {
	case <-s.started:
		return errors.New("app: server already started")
	default:
	}
	if len(servlets) == 0 {
		return nil
	}
	for _, servlet := range servlets {
		for _, used := range s.servlets {
			if used.Name() == servlet.Name() {
				return fmt.Errorf(`app: servlet "%s" already registered`, servlet.Name())
			}
		}
	}
	s.servlets = append(s.servlets, servlets...)
	return nil
}

func (s *SimpleApp) Start() error {
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
	call(s.sortServlets)    // 按优先级排序
	call(s.initServlets)    // 初始化网络组件
	call(s.configureKernel) // 配置应用
	call(s.configureRoutes) // 注册路由
	call(s.bootServlets)    // 启动网络组件
	call(s.bootstrap)       // 启动网络服务器
	return err
}

func (s *SimpleApp) sortServlets() error {
	if s.servlets.Len() == 0 {
		return errors.New("no registers servlet")
	}
	sort.Sort(s.servlets)
	return nil
}

func (s *SimpleApp) initServlets() error {
	if s.inits == nil {
		s.inits = make(map[string]*servletInitContext)
	} else {
		clear(s.inits)
	}
	sort.Sort(s.servlets)
	for _, servlet := range s.servlets {
		ctx := NewServletInitContext().(*servletInitContext)
		s.inits[servlet.Name()] = ctx
		err := servlet.Init(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SimpleApp) configureKernel() error {
	kernel := slim.New()
	kernel.Debug = !env.IsEnv("prod")
	kernel.Logger = log.Default()
	kernel.Use(slim.Recover())
	kernel.Use(middleware.LoggingWithConfig(middleware.LoggingConfig{
		DisableRequestID:     s.options.disableRequestID,
		KeyedPrefixInContext: s.options.keyedLogging,
	}))
	s.slim = kernel
	return nil
}

func (s *SimpleApp) configureRoutes() error {
	// 按 Servlet 的顺序注册路由
	for _, servlet := range s.servlets {
		ctx := s.inits[servlet.Name()]
		for _, route := range ctx.routes {
			route(s.slim)
		}
	}
	return nil
}

func (s *SimpleApp) bootServlets() error {
	for _, servlet := range s.servlets {
		err := servlet.Bootstrap()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SimpleApp) bootstrap() error {
	addr := env.String("SERVER_ADDR", "0.0.0.0")
	port := env.Int("SERVER:PORT", 1234)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}
	listener = net.Listener(TCPKeepAliveListener{
		TCPListener: listener.(*net.TCPListener),
	})
	httpServer := &http.Server{
		Handler:        s.slim,
		MaxHeaderBytes: s.options.maxHeaderBytes,
		ReadTimeout:    s.options.readTimeout,
		WriteTimeout:   s.options.writeTimeout,
		IdleTimeout:    s.options.idleTimeout,
	}
	go func() {
		if srvErr := httpServer.Serve(listener); srvErr != nil {
			log.Error("encountered an error while serving listener: ", srvErr)
		}
	}()
	log.Info("Listening on %s", listener.Addr().String())
	// 监听停止命令，停止网络服务
	go func() {
		errChan := <-s.exit
		// 销毁注册的服务
		for i := len(s.servlets) - 1; i >= 0; i-- {
			if ex := s.servlets[i].Destroy(); ex != nil {
				log.Warn("servlet Stop returned with error: ", ex)
			}
		}
		// stop the listener
		errChan <- listener.Close()
	}()
	return nil
}

func (s *SimpleApp) Stop() error {
	select {
	case <-s.started:
		exit := make(chan error)
		s.exit <- exit
		return <-exit
	default:
		return nil
	}
}

// TCPKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g., closing laptop mid-download) eventually
// go away.
//
// This is here because it is not exposed in the stdlib and
// we'd prefer to have a hold of the http.Server's net.Listener so we can close it
// on shutdown.
//
// Taken from here: https://golang.org/src/net/http/server.go?s=63121:63175#L2120
type TCPKeepAliveListener struct {
	*net.TCPListener
}

// Accept accepts the next incoming call and returns the new
// connection. KeepAlivePeriod is set properly.
func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return
	}
	err = tc.SetKeepAlivePeriod(3 * time.Minute)
	if err != nil {
		return
	}
	return tc, nil
}
