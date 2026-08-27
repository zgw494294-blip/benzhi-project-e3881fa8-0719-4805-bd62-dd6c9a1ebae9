package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/store/sqlite"
	"caption-release-gate/internal/web"
	"caption-release-gate/internal/workflow"
)

type application struct {
	config   config
	listener net.Listener
	server   *http.Server
	workflow *workflow.Service
}

func buildApplication(ctx context.Context, cfg config) (*application, error) {
	repository, err := sqlite.Open(ctx, cfg.databasePath)
	if err != nil {
		return nil, err
	}
	auditService, err := audit.NewService(cfg.issuer, []byte(cfg.secret))
	if err != nil {
		repository.Close()
		return nil, err
	}
	workflowService := workflow.NewService(repository, auditService)
	handler := web.NewHandler(workflowService)
	server := &http.Server{
		Addr:              cfg.addr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return &application{config: cfg, server: server, workflow: workflowService}, nil
}

func (a *application) listen() error {
	listener, err := net.Listen("tcp", a.config.addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", a.config.addr, err)
	}
	a.listener = listener
	return nil
}

func (a *application) serve(errorChannel chan<- error) {
	go func() {
		slog.Info("字幕发布准入工作台已启动", "addr", a.listener.Addr().String())
		err := a.server.Serve(a.listener)
		if err == http.ErrServerClosed {
			err = nil
		}
		errorChannel <- err
	}()
}

func (a *application) shutdown(ctx context.Context) error {
	serverErr := a.server.Shutdown(ctx)
	storeErr := a.workflow.Close()
	if serverErr != nil {
		return serverErr
	}
	return storeErr
}
