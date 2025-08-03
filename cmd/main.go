// main.go
package main

import (
	"cardano-tx-sync/config"
	"cardano-tx-sync/internal/api"
	"cardano-tx-sync/internal/chainsync"
	"cardano-tx-sync/internal/handler"
	"cardano-tx-sync/internal/kafka"
	"cardano-tx-sync/internal/storage"
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/SundaeSwap-finance/ogmigo"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		logger.Fatal("cannot load config", zap.Error(err))
	}

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down...", zap.String("signal", sig.String()))
		cancel()
	}()

	// Initialize storage
	db, err := storage.NewPostgresStorage(cfg.DB)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	logger.Info("database connection established")

	// Initialize Kafka producer
	producer, err := kafka.NewProducer(cfg.Kafka.Brokers)
	if err != nil {
		logger.Fatal("failed to create kafka producer", zap.Error(err))
	}
	defer producer.Close()
	logger.Info("kafka producer created")

	// Initialize Ogmigo client
	ogmigoClient := ogmigo.New(
		ogmigo.WithEndpoint(cfg.Ogmios.Endpoint),
		// ogmigo.WithLogger(zap.NewNop().Sugar()), // Replace with a proper logger if needed
	)

	// Initialize block handler
	blockHandler := handler.NewBlockHandler(db, producer, logger)

	// Initialize ChainSync service
	syncer := chainsync.NewSyncer(ogmigoClient, blockHandler, db, logger, cfg.ChainSync)

	// Start the ChainSync service in a separate goroutine
	go func() {
		if err := syncer.Start(ctx); err != nil {
			logger.Error("chainsync service stopped", zap.Error(err))
		}
	}()

	// Initialize and start API server
	apiServer := api.NewServer(db, syncer, logger)
	go func() {
		if err := apiServer.Start(cfg.API.ListenAddress); err != nil {
			logger.Error("api server failed to start", zap.Error(err))
		}
	}()

	logger.Info("application started successfully")

	// Wait for context to be cancelled (due to signal or other error)
	<-ctx.Done()

	logger.Info("application shutting down")
}
