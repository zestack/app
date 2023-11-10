package app

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
	"zestack.dev/env"
	"zestack.dev/slim"
)

// ServerConfig 服务器配置
// TODO(hupeh): 支持 HTTP2
type ServerConfig struct {
	Addr net.Addr
	// MaxHeaderBytes is used by the http server to limit the size of request headers.
	// This may need to be increased if accepting cookies from the public.
	MaxHeaderBytes int
	// ReadTimeout is used by the http server to set a maximum duration before
	// timing out read of the request. The default timeout is 10 seconds.
	ReadTimeout time.Duration
	// WriteTimeout is used by the http server to set a maximum duration before
	// timing out write of the response. The default timeout is 10 seconds.
	WriteTimeout time.Duration
	// IdleTimeout is used by the http server to set a maximum duration for
	// keep-alive connections.
	IdleTimeout time.Duration
	// TLSConfig optionally provides a TLS configuration for use
	// by ServeTLS and ListenAndServeTLS. Note that this value is
	// cloned by ServeTLS and ListenAndServeTLS, so it's not
	// possible to modify the configuration with methods like
	// tls.Config.SetSessionTicketKeys. To use
	// SetSessionTicketKeys, use Server.Serve with a TLS Listener
	// instead.
	TLSConfig *tls.Config
	// MultipartMemoryLimit 文件上传大小限制
	MultipartMemoryLimit int64
}

func (s *ServerConfig) use(kernel *slim.Slim) {
	if s.MultipartMemoryLimit > 0 {
		kernel.MultipartMemoryLimit = s.MultipartMemoryLimit
	}
}

// ensure 确保服务器配置正确，如果未设置则加载环境变量
func (s *ServerConfig) ensure() (err error) {
	if s.Addr == nil {
		addr := env.String("SERVER_ADDR", "0.0.0.0")
		port := env.Int("SERVER_PORT", 1234)
		s.Addr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", addr, port))
		if err != nil {
			return
		}
	}
	if s.MaxHeaderBytes <= 0 {
		s.MaxHeaderBytes = env.Int("SERVER_MAX_HEADER_BYTES", http.DefaultMaxHeaderBytes)
	}
	if s.WriteTimeout <= 0 {
		s.WriteTimeout = env.Duration("SERVER_WRITE_TIMEOUT")
	}
	if s.ReadTimeout <= 0 {
		s.ReadTimeout = env.Duration("SERVER_READ_TIMEOUT")
	}
	if s.IdleTimeout <= 0 {
		s.IdleTimeout = env.Duration("SERVER_IDLE_TIMEOUT")
	}
	return
}

func newListener(app *simpleApp) (net.Listener, error) {
	config := app.config.Server
	addr := config.Addr.String()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	listener = net.Listener(TCPKeepAliveListener{
		TCPListener: listener.(*net.TCPListener),
	})
	return listener, nil
}

func newServer(app *simpleApp) *http.Server {
	config := app.config.Server
	return &http.Server{
		Handler:        app.slim,
		MaxHeaderBytes: config.MaxHeaderBytes,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		TLSConfig:      config.TLSConfig,
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
