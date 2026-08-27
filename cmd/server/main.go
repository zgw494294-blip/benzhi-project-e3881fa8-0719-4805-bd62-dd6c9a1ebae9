package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	rootContext, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	app, err := buildApplication(rootContext, cfg)
	if err != nil {
		return err
	}
	if err := app.listen(); err != nil {
		app.workflow.Close()
		return err
	}
	serveErrors := make(chan error, 1)
	app.serve(serveErrors)
	if cfg.selfcheck {
		selfcheckContext, stop := context.WithTimeout(rootContext, cfg.selfcheckTimeout)
		checkErr := runSelfcheck(selfcheckContext, app.listener.Addr().String())
		stop()
		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		shutdownErr := app.shutdown(shutdownContext)
		shutdownCancel()
		serveErr := <-serveErrors
		if checkErr != nil {
			return fmt.Errorf("selfcheck 失败: %w", checkErr)
		}
		if shutdownErr != nil {
			return fmt.Errorf("selfcheck 关闭失败: %w", shutdownErr)
		}
		if serveErr != nil {
			return serveErr
		}
		slog.Info("selfcheck 通过", "addr", cfg.addr)
		return nil
	}
	select {
	case err := <-serveErrors:
		if err != nil {
			app.workflow.Close()
			return err
		}
		return app.workflow.Close()
	case <-rootContext.Done():
		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := app.shutdown(shutdownContext); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}
}
