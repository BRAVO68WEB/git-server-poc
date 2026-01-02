package router

import (
	"github.com/bravo68web/stasis/internal/injectable"
	"github.com/bravo68web/stasis/internal/server"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
)

type Router struct {
	server *server.Server
	Deps   *injectable.Dependencies
}

// NewRouter creates a new Router instance.
func NewRouter(s *server.Server) *Router {
	deps := injectable.LoadDependencies(s.Config, s.DB)

	return &Router{
		server: s,
		Deps:   &deps,
	}
}

// RegisterRoutes sets up the routes and middleware for the server.
func (r *Router) RegisterRoutes() {
	// Get allowed origins from config, default to localhost for development
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}

	// Add configured frontend URL if available (from OIDC config)
	if r.server.Config.OIDC.FrontendURL != "" {
		allowedOrigins = append(allowedOrigins, r.server.Config.OIDC.FrontendURL)
	}

	// Setup logging and recovery middleware
	r.setupHTTPLoggerAndRecovery()

	// Apply CORS middleware
	r.server.Use(middleware.CORSMiddleware(allowedOrigins))

	r.docsRouter()

	r.healthRouter()
	r.authRouter()
	r.repoRouter()
	r.gitRouter()
	r.sshKeyRouter()
	r.tokenRouter()
	r.ciRouter()
}

func (r *Router) setupHTTPLoggerAndRecovery() {
	// Add custom logging middleware
	loggerMiddlewareCfg := &middleware.LoggerConfig{
		Logger:           r.server.Logger,
		SkipPaths:        []string{"/health", "/healthz", "/ready", "/readyz", "/metrics"},
		SkipPathPrefixes: []string{"/static/"},
		LogRequestBody:   r.server.Config.Logging.Development,
		LogResponseBody:  false,
		MaxBodyLogSize:   1024,
		TraceIDHeader:    "X-Trace-ID",
		RequestIDHeader:  "X-Request-ID",
		IncludeHeaders:   r.server.Config.Logging.Development,
		SensitiveHeaders: []string{"Authorization", "Cookie", "X-API-Key", "X-Auth-Token"},
	}
	r.server.Use(middleware.LoggerMiddlewareWithConfig(loggerMiddlewareCfg))

	// Add recovery middleware with logging
	recoveryCfg := &middleware.RecoveryConfig{
		Logger:           r.server.Logger,
		EnableStackTrace: r.server.Config.Logging.Development || r.server.Config.Server.Mode != "release",
		StackTraceSize:   4096,
	}
	r.server.Use(middleware.RecoveryMiddlewareWithConfig(recoveryCfg))
}
