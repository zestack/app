package app

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"zestack.dev/slim"
)

// CORSConfig defines the config for CORS middleware.
type CORSConfig struct {
	// AllowOrigin defines a list of origins that may access the resource.
	// Optional. Default value []string{"*"}.
	AllowOrigins []string

	// AllowOriginFunc is a custom function to validate the origin. It takes the
	// origin as an argument and returns true if allowed or false otherwise. If
	// an error is returned, it is returned by the handler. If this option is
	// set, AllowOrigins is ignored.
	// Optional.
	AllowOriginFunc func(origin string) (bool, error)

	// AllowMethods defines a list methods allowed when accessing the resource.
	// This is used in response to a preflight request.
	// Optional. Default value DefaultCORSConfig.AllowMethods.
	AllowMethods []string

	// AllowHeaders defines a list of request headers that can be used when
	// making the actual request. This is in response to a preflight request.
	// Optional. Default value []string{}.
	AllowHeaders []string

	// AllowCredentials indicates whether or not the response to the request
	// can be exposed when the credential flag is true. When used as part of
	// a response to a preflight request, this indicates whether or not the
	// actual request can be made using credentials.
	// Optional. Default value is false.
	AllowCredentials bool

	// ExposeHeaders defines the whitelist headers that clients are allowed to
	// access.
	// Optional. Default value []string{}.
	ExposeHeaders []string

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached.
	// Optional. Default value 0.
	MaxAge int
}

func cors(config CORSConfig) slim.MiddlewareFunc {
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = []string{"*"}
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
		}
	}

	var allowOriginPatterns []string
	for _, origin := range config.AllowOrigins {
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.Replace(pattern, "\\*", ".*", -1)
		pattern = strings.Replace(pattern, "\\?", ".", -1)
		pattern = "^" + pattern + "$"
		allowOriginPatterns = append(allowOriginPatterns, pattern)
	}

	allowMethods := strings.Join(config.AllowMethods, ",")
	allowHeaders := strings.Join(config.AllowHeaders, ",")
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")
	maxAge := strconv.Itoa(config.MaxAge)

	return func(c slim.Context, next slim.HandlerFunc) error {
		req := c.Request()
		res := c.Response()
		origin := req.Header.Get(slim.HeaderOrigin)
		allowOrigin := ""

		preflight := req.Method == http.MethodOptions
		res.Header().Add(slim.HeaderVary, slim.HeaderOrigin)

		// No Origin provided
		if origin == "" {
			if !preflight {
				return next(c)
			}
			return c.NoContent(http.StatusNoContent)
		}

		if config.AllowOriginFunc != nil {
			allowed, err := config.AllowOriginFunc(origin)
			if err != nil {
				return err
			}
			if allowed {
				allowOrigin = origin
			}
		} else {
			// Check allowed origins
			for _, o := range config.AllowOrigins {
				if o == "*" && config.AllowCredentials {
					allowOrigin = origin
					break
				}
				if o == "*" || o == origin {
					allowOrigin = o
					break
				}
				if matchSubdomain(origin, o) {
					allowOrigin = origin
					break
				}
			}

			// Check allowed origin patterns
			for _, re := range allowOriginPatterns {
				if allowOrigin == "" {
					didx := strings.Index(origin, "://")
					if didx == -1 {
						continue
					}
					domAuth := origin[didx+3:]
					// to avoid regex cost by invalid long domain
					if len(domAuth) > 253 {
						break
					}

					if match, _ := regexp.MatchString(re, origin); match {
						allowOrigin = origin
						break
					}
				}
			}
		}

		// Origin isn't allowed
		if allowOrigin == "" {
			if !preflight {
				return next(c)
			}
			return c.NoContent(http.StatusNoContent)
		}

		// Simple request
		if !preflight {
			res.Header().Set(slim.HeaderAccessControlAllowOrigin, allowOrigin)
			if config.AllowCredentials {
				res.Header().Set(slim.HeaderAccessControlAllowCredentials, "true")
			}
			if exposeHeaders != "" {
				res.Header().Set(slim.HeaderAccessControlExposeHeaders, exposeHeaders)
			}
			return next(c)
		}

		// Preflight request
		res.Header().Add(slim.HeaderVary, slim.HeaderAccessControlRequestMethod)
		res.Header().Add(slim.HeaderVary, slim.HeaderAccessControlRequestHeaders)
		res.Header().Set(slim.HeaderAccessControlAllowOrigin, allowOrigin)
		res.Header().Set(slim.HeaderAccessControlAllowMethods, allowMethods)
		if config.AllowCredentials {
			res.Header().Set(slim.HeaderAccessControlAllowCredentials, "true")
		}
		if allowHeaders != "" {
			res.Header().Set(slim.HeaderAccessControlAllowHeaders, allowHeaders)
		} else {
			h := req.Header.Get(slim.HeaderAccessControlRequestHeaders)
			if h != "" {
				res.Header().Set(slim.HeaderAccessControlAllowHeaders, h)
			}
		}
		if config.MaxAge > 0 {
			res.Header().Set(slim.HeaderAccessControlMaxAge, maxAge)
		}
		return c.NoContent(http.StatusNoContent)
	}
}

func matchScheme(domain, pattern string) bool {
	didx := strings.Index(domain, ":")
	pidx := strings.Index(pattern, ":")
	return didx != -1 && pidx != -1 && domain[:didx] == pattern[:pidx]
}

// matchSubdomain compares authority with wildcard
func matchSubdomain(domain, pattern string) bool {
	if !matchScheme(domain, pattern) {
		return false
	}
	didx := strings.Index(domain, "://")
	pidx := strings.Index(pattern, "://")
	if didx == -1 || pidx == -1 {
		return false
	}
	domAuth := domain[didx+3:]
	// to avoid long loop by invalid long domain
	if len(domAuth) > 253 {
		return false
	}
	patAuth := pattern[pidx+3:]

	domComp := strings.Split(domAuth, ".")
	patComp := strings.Split(patAuth, ".")
	for i := len(domComp)/2 - 1; i >= 0; i-- {
		opp := len(domComp) - 1 - i
		domComp[i], domComp[opp] = domComp[opp], domComp[i]
	}
	for i := len(patComp)/2 - 1; i >= 0; i-- {
		opp := len(patComp) - 1 - i
		patComp[i], patComp[opp] = patComp[opp], patComp[i]
	}

	for i, v := range domComp {
		if len(patComp) <= i {
			return false
		}
		p := patComp[i]
		if p == "*" {
			return true
		}
		if p != v {
			return false
		}
	}
	return false
}
