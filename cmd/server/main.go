package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/charmbracelet/ssh"

	"github.com/bravo68web/stasis/internal/injectable"
	"github.com/bravo68web/stasis/internal/server"
	"github.com/bravo68web/stasis/internal/transport/http/router"
	sshserver "github.com/bravo68web/stasis/internal/transport/ssh"
	"github.com/bravo68web/stasis/pkg/logger"
)

func main() {
	// Initialize server (this also initializes the logger)
	s := server.New()
	log := s.Logger

	log.Info("Starting Stasis Git Server",
		logger.String("version", "1.0.0"),
		logger.Bool("development", s.Config.Logging.Development),
	)

	// Create router and register routes
	r := router.NewRouter(s)
	r.RegisterRoutes()

	log.Info("Routes registered successfully")

	// Create a channel for shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		addr := ":" + strconv.Itoa(s.Config.Server.Port)
		log.Info("Starting HTTP server",
			logger.String("address", addr),
			logger.String("mode", s.Config.Server.Mode),
		)
		if err := s.Run(addr); err != nil {
			log.Error("HTTP server error",
				logger.Error(err),
				logger.String("address", addr),
			)
		}
	}()

	// Start SSH server if enabled
	var sshSrv *sshserver.Server
	if s.Config.SSH.Enabled {
		log.Info("SSH server is enabled, initializing...",
			logger.String("address", s.Config.SSH.Address()),
			logger.String("host_key_path", s.Config.SSH.HostKeyPath),
		)

		// Load dependencies for SSH server
		deps := injectable.LoadDependencies(s.Config, s.DB)

		var err error
		sshSrv, err = sshserver.NewServer(
			&s.Config.SSH,
			&s.Config.Storage,
			deps.AuthService,
			deps.RepoService,
			deps.CIService,
			deps.GitService,
			deps.Storage,
		)
		if err != nil {
			log.Error("Failed to create SSH server",
				logger.Error(err),
			)
		} else {
			go func() {
				log.Info("Starting SSH server",
					logger.String("address", s.Config.SSH.Address()),
				)
				if err := sshSrv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
					log.Error("SSH server error",
						logger.Error(err),
					)
				}
			}()
		}
	} else {
		log.Info("SSH server is disabled")
	}

	log.Info("Servers started successfully. Press Ctrl+C to shutdown.",
		logger.Int("http_port", s.Config.Server.Port),
		logger.Bool("ssh_enabled", s.Config.SSH.Enabled),
	)

	// Wait for shutdown signal
	sig := <-done
	log.Info("Received shutdown signal",
		logger.String("signal", sig.String()),
	)

	log.Info("Initiating graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown SSH server if running
	if sshSrv != nil {
		log.Info("Shutting down SSH server...")
		if err := sshSrv.Shutdown(shutdownCtx); err != nil {
			log.Error("SSH server shutdown error",
				logger.Error(err),
			)
		} else {
			log.Info("SSH server shutdown complete")
		}
	}

	// Close server resources (including logger)
	if err := s.Close(); err != nil {
		log.Error("Error closing server resources",
			logger.Error(err),
		)
	}

	log.Info("Server shutdown complete. Goodbye!")

	// Final sync before exit
	_ = logger.SyncGlobal()
}
