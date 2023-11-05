package middleware

import (
	stdctx "context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/xid"
	"zestack.dev/log"
	"zestack.dev/slim"
)

type LoggingConfig struct {
	DisableRequestID bool
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
		l.Info("Started %s %s for %s", c.Request().Method, c.RequestURI(), c.RealIP())
		c.SetRequest(c.Request().WithContext(stdctx.WithValue(c, "logger", l)))
		c.SetLogger(l)
		if err = next(c); err != nil {
			c.Error(err)
		}
		stop := time.Now()
		status := c.Response().Status()
		content := fmt.Sprintf(
			"Completed %s %s %v %s in %v",
			c.Request().Method, c.RequestURI(), status,
			http.StatusText(c.Response().Status()), stop.Sub(start),
		)
		// TODO(hupeh): 打印颜色
		//if status >= 500 {
		//	content = color.Cyan(content)
		//} else if status >= 400 {
		//	content = color.Red(content)
		//} else if status >= 300 {
		//	if status == 304 {
		//		content = color.Yellow(content)
		//	} else {
		//		content = color.White(content)
		//	}
		//} else if status >= 200 {
		//	content = color.Green(content)
		//}
		l.Info(content)
		return
	}
}
