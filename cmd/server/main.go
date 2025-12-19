package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/charmbracelet/ssh"

	"github.com/bravo68web/githut/internal/injectable"
	"github.com/bravo68web/githut/internal/server"
	"github.com/bravo68web/githut/internal/transport/http/router"
	sshserver "github.com/bravo68web/githut/internal/transport/ssh"
)

func main() {
	s := server.New()

	// Run database migrations (including OIDC migration)
	if err := s.DB.RunMigrations(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Create router and register routes
	r := router.NewRouter(s)
	r.RegisterRoutes()

	// Create a channel for shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		addr := ":" + strconv.Itoa(s.Config.Server.Port)
		log.Printf("Starting HTTP server on %s", addr)
		if err := s.Run(addr); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start SSH server if enabled
	if s.Config.SSH.Enabled {
		// Load dependencies for SSH server
		deps := injectable.LoadDependencies(s.Config, s.DB)

		sshSrv, err := sshserver.NewServer(
			&s.Config.SSH,
			&s.Config.Storage,
			deps.AuthService,
			deps.RepoService,
			deps.Storage,
		)
		if err != nil {
			log.Printf("Failed to create SSH server: %v", err)
		} else {
			go func() {
				log.Printf("Starting SSH server on %s", s.Config.SSH.Address())
				if err := sshSrv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
					log.Printf("SSH server error: %v", err)
				}
			}()

			// Handle graceful shutdown for SSH server
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := sshSrv.Shutdown(ctx); err != nil {
					log.Printf("SSH server shutdown error: %v", err)
				}
			}()
		}
	}

	log.Println("Servers started. Press Ctrl+C to shutdown.")

	// Wait for shutdown signal
	<-done
	log.Println("Shutting down servers...")
}
