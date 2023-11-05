package app

import (
	"crypto/tls"
	"time"
)

type Options struct {
	// maxHeaderBytes is used by the http server to limit the size of request headers.
	// This may need to be increased if accepting cookies from the public.
	maxHeaderBytes int
	// readTimeout is used by the http server to set a maximum duration before
	// timing out read of the request. The default timeout is 10 seconds.
	readTimeout time.Duration
	// writeTimeout is used by the http server to set a maximum duration before
	// timing out write of the response. The default timeout is 10 seconds.
	writeTimeout time.Duration
	// idleTimeout is used by the http server to set a maximum duration for
	// keep-alive connections.
	idleTimeout time.Duration
	// tlsConfig optionally provides a TLS configuration for use
	// by ServeTLS and ListenAndServeTLS. Note that this value is
	// cloned by ServeTLS and ListenAndServeTLS, so it's not
	// possible to modify the configuration with methods like
	// tls.Config.SetSessionTicketKeys. To use
	// SetSessionTicketKeys, use Server.Serve with a TLS Listener
	// instead.
	tlsConfig *tls.Config
}

type Option func(o *Options)

func MaxHeaderBytes(maxHeaderBytes int) Option {
	return func(o *Options) {
		o.maxHeaderBytes = maxHeaderBytes
	}
}

func ReadTimeout(readTimeout time.Duration) Option {
	return func(o *Options) {
		o.readTimeout = readTimeout
	}
}

func WriteTimeout(writeTimeout time.Duration) Option {
	return func(o *Options) {
		o.writeTimeout = writeTimeout
	}
}

func IdleTimeout(idleTimeout time.Duration) Option {
	return func(o *Options) {
		o.idleTimeout = idleTimeout
	}
}

func TLSConfig(tlsConfig *tls.Config) Option {
	return func(o *Options) {
		o.tlsConfig = tlsConfig
	}
}
