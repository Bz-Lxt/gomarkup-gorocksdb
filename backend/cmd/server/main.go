package main

import (
	"os"
	"os/signal"
	"syscall"

	"gorocksdb/internal/api"
	"gorocksdb/internal/logger"
	"gorocksdb/pkg/gorocksdb"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	logger.Init(env("GOROCKSDB_LOG_LEVEL", "info"))
	dir := env("GOROCKSDB_DATA", "./data")
	profile := env("GOROCKSDB_PROFILE", "demo")
	addr := env("GOROCKSDB_LISTEN", ":8080")
	cors := env("GOROCKSDB_CORS_ORIGINS", "http://localhost:28741")

	db, err := gorocksdb.Open(gorocksdb.Options{
		Dir:     dir,
		Profile: profile,
		Sync:    os.Getenv("GOROCKSDB_SYNC") == "1",
	})
	if err != nil {
		logger.L().Error("open failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	srv := api.New(db, cors)
	go func() {
		if err := srv.Listen(addr); err != nil {
			logger.L().Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.L().Info("shutting down")
}
