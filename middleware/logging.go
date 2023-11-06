package middleware

import (
	stdctx "context"
	"fmt"
	"github.com/rs/xid"
	"net/http"
	"time"
	"zestack.dev/log"
	"zestack.dev/slim"
)

type color int

var (
	cyan   = color(96)
	red    = color(91)
	yellow = color(93)
	white  = color(97)
	green  = color(92)
)

func (u color) wrap(s string) string {
	if u < 91 {
		return s
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", u, s)
}

// LoggingConfig 日志打印配置
type LoggingConfig struct {
	// DisableRequestID 是否开启 RequestID
	DisableRequestID bool
	// KeyedPrefixInContext 自定义的关联前缀的日志实例到请求上下文中，比如：
	//
	//   LoggingConfig{
	//     DisableRequestID: false,
	//     DisableRequestID: map[string]string{
	//       "db:logger":    "db",    // 将数据库操作与请求关联
	//       "redis:logger": "redis", // 将 redis 操作与请求关联
	//       //...其它关联
	//     }
	//   }
	KeyedPrefixInContext map[string]string
}

var DefaultLoggingConfig = LoggingConfig{
	DisableRequestID: false,
}

func LoggingWithConfig(config LoggingConfig) slim.MiddlewareFunc {
	return config.ToMiddleware()
}

func Logging() slim.MiddlewareFunc {
	return LoggingWithConfig(DefaultLoggingConfig)
}

func requestId(c slim.Context) string {
	id := c.Header(slim.HeaderXRequestID)
	if id == "" {
		id = xid.New().String()
		c.SetHeader(slim.HeaderXRequestID, id)
	}
	return id
}

func (config LoggingConfig) ToMiddleware() slim.MiddlewareFunc {
	return func(c slim.Context, next slim.HandlerFunc) (err error) {
		start := time.Now()
		l := log.Default()
		if !config.DisableRequestID {
			id := requestId(c)
			l = l.With(log.String("id", id))
		}
		if len(config.KeyedPrefixInContext) > 0 {
			ctx := c.Request().Context()
			for key, prefix := range config.KeyedPrefixInContext {
				ctx = stdctx.WithValue(ctx, key, l.WithPrefix(prefix))
			}
			c.SetRequest(c.Request().WithContext(ctx))
		}
		l.Info("Started %s %s for %s", c.Request().Method, c.RequestURI(), c.RealIP())
		c.SetRequest(c.Request().WithContext(stdctx.WithValue(c, "logger", l)))
		c.SetLogger(l)
		if err = next(c); err != nil {
			c.Error(err)
		}
		stop := time.Now()
		status := c.Response().Status()
		content := fmt.Sprintf(
			"Completed %s %s %v %s in %s",
			c.Request().Method,
			c.RequestURI(),
			status,
			http.StatusText(c.Response().Status()),
			stop.Sub(start).String(),
		)
		var colorize color
		if w, ok := l.Output().(*log.Writer); ok && w.IsColorful() {
			if status >= 500 {
				colorize = cyan
			} else if status >= 400 {
				colorize = red
			} else if status >= 300 {
				if status == 304 {
					colorize = yellow
				} else {
					colorize = white
				}
			} else if status >= 200 {
				colorize = green
			}
		}
		l.Info(colorize.wrap(content))
		return
	}
}
