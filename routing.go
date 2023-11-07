package app

import (
	"io/fs"

	"zestack.dev/slim"
)

// RoutingConfig 路由配置
type RoutingConfig struct {
	ErrorHandler         slim.ErrorHandlerFunc
	Validator            slim.Validator
	Renderer             slim.Renderer
	Filesystem           fs.FS
	JSONSerializer       slim.Serializer
	XMLSerializer        slim.Serializer
	MultipartMemoryLimit int64  // 文件上传大小限制
	PrettyIndent         string // json/xml 格式化缩进
	JSONPCallbacks       []string
	Premiddleware        []slim.MiddlewareFunc
}

func (c RoutingConfig) use(s *slim.Slim) {
	if c.ErrorHandler != nil {
		s.ErrorHandler = c.ErrorHandler
	}
	if c.Validator != nil {
		s.Validator = c.Validator
	}
	if c.Renderer != nil {
		s.Renderer = c.Renderer
	}
	if c.Filesystem != nil {
		s.Filesystem = c.Filesystem
	}
	if c.JSONSerializer != nil {
		s.JSONSerializer = c.JSONSerializer
	}
	if c.XMLSerializer != nil {
		s.XMLSerializer = c.XMLSerializer
	}
	if c.MultipartMemoryLimit > 0 {
		s.MultipartMemoryLimit = c.MultipartMemoryLimit
	}
	if len(c.JSONPCallbacks) > 0 {
		s.JSONPCallbacks = c.JSONPCallbacks[:]
	}
	if len(c.Premiddleware) > 0 {
		s.Use(c.Premiddleware...)
	}
}
