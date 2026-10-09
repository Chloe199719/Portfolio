package main

import (
	assets "chloe.dev/home/backend"
	"chloe.dev/home/backend/internal/platform"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c, e := platform.LoadConfig()
	if e != nil {
		slog.Error("configuration", "error", e)
		os.Exit(1)
	}
	s, e := platform.New(ctx, c, assets.Assets)
	if e != nil {
		slog.Error("startup", "error", e)
		os.Exit(1)
	}
	defer s.DB.Close()
	if len(os.Args) > 1 {
		if e = s.Command(ctx, os.Args[1:]); e != nil {
			slog.Error("command", "error", e)
			os.Exit(1)
		}
		return
	}
	go s.RunJobs(ctx)
	srv := &http.Server{Addr: c.Addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 16384}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	slog.Info("backend listening", "address", c.Addr)
	if e = srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		slog.Error("server", "error", e)
		os.Exit(1)
	}
}
