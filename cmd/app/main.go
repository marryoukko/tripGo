package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tripgo.local/internal/config"
	api "tripgo.local/internal/generated"
	"tripgo.local/internal/handler"
	"tripgo.local/internal/repository"
	"tripgo.local/internal/service"
	"tripgo.local/internal/transaction"
)

func main() {
	// 1. Конфигурация из env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// 2. Пул соединений pgxpool
	pool, err := newPool(cfg)
	if err != nil {
		log.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	// 3. Проверка доступности БД — без неё не стартуем
	pingCtx, cancel := context.WithTimeout(context.Background(), cfg.DBConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	log.Println("db connected")

	// 4. Сборка зависимостей (без DI-контейнера, всё руками)
	txManager := transaction.NewManager(pool)
	tripRepo := repository.NewTripRepository(pool)
	historyRepo := repository.NewHistoryRepository(pool)
	tripSvc := service.NewTripService(tripRepo, historyRepo, txManager)
	h := handler.New(tripSvc, pool)

	// 5. Роутер: маршруты из OpenAPI + /health и /ready
	r := chi.NewRouter()
	api.HandlerFromMux(h, r)

	// 6. HTTP-сервер с таймаутами
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 7. Graceful shutdown по SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("stopped")
}

func newPool(cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.MinConns = cfg.DBMinConns
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.ConnConfig.ConnectTimeout = cfg.DBConnectTimeout
	return pgxpool.NewWithConfig(context.Background(), poolCfg)
}
