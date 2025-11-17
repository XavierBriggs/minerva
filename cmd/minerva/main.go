package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fortuna/minerva/internal/api/rest"
	"github.com/fortuna/minerva/internal/api/websocket"
	"github.com/fortuna/minerva/internal/backfill"
	"github.com/fortuna/minerva/internal/cache"
	"github.com/fortuna/minerva/internal/publisher"
	"github.com/fortuna/minerva/internal/scheduler"
	"github.com/fortuna/minerva/internal/store"
)

const (
	serviceName    = "minerva"
	serviceVersion = "2.0.0"
)

func main() {
	log.Printf("Starting %s v%s - Sports Analytics Service", serviceName, serviceVersion)

	// Load configuration from environment
	config := loadConfig()

	// Initialize database connection
	db, err := store.NewDatabase(config.AtlasDSN)
	if err != nil {
		log.Fatalf("Failed to connect to Atlas database: %v", err)
	}
	defer db.Close()

	log.Println("✓ Connected to Atlas database")

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("✓ Database migrations applied")

	// Seed initial data (non-fatal - may already exist)
	if err := db.SeedData(); err != nil {
		log.Printf("⚠️  Seed data warning: %v (continuing anyway)", err)
	} else {
		log.Println("✓ Seed data applied")
	}

	// Initialize Redis client with retry logic
	var redisCache *cache.RedisCache
	maxRetries := 30
	retryDelay := 2 * time.Second
	
	log.Println("Connecting to Redis...")
	for i := 0; i < maxRetries; i++ {
		redisCache, err = cache.NewRedisCache(config.RedisURL)
		if err == nil {
			break
		}
		
		if i < maxRetries-1 {
			log.Printf("Redis connection attempt %d/%d failed: %v (retrying in %v)", i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		} else {
			log.Fatalf("Failed to connect to Redis after %d attempts: %v", maxRetries, err)
		}
	}
	defer redisCache.Close()

	log.Println("✓ Connected to Redis")

	// Initialize Redis publisher with retry logic
	var redisPublisher *publisher.RedisPublisher
	log.Println("Initializing Redis publisher...")
	for i := 0; i < maxRetries; i++ {
		redisPublisher, err = publisher.NewRedisPublisher(config.RedisURL)
		if err == nil {
			break
		}
		
		if i < maxRetries-1 {
			log.Printf("Redis publisher attempt %d/%d failed: %v (retrying in %v)", i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		} else {
			log.Fatalf("Failed to initialize Redis publisher after %d attempts: %v", maxRetries, err)
		}
	}
	defer redisPublisher.Close()

	log.Println("✓ Redis publisher initialized")

	// Initialize scheduler/orchestrator with configuration
	schedulerConfig := &scheduler.Config{
		LivePollInterval:     10 * time.Second,
		DailyIngestionHour:   3,
		CurrentSeasonID:      getEnv("CURRENT_SEASON", "2024-25"),
		EnableLivePolling:    getEnv("ENABLE_LIVE_POLLING", "true") == "true",
		EnableDailyIngestion: getEnv("ENABLE_DAILY_INGESTION", "true") == "true",
		MaxRetries:           3,
		RetryDelay:           5 * time.Second,
	}
	
	sched, err := scheduler.NewOrchestrator(db, redisCache, redisPublisher, schedulerConfig)
	if err != nil {
		log.Fatalf("Failed to create scheduler: %v", err)
	}
	
	// Start scheduler in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Start(ctx)

	log.Println("✓ Scheduler started")

	// Initialize backfill service
	backfillService := backfill.NewService(db, config.ESPNAPIBase, log.Default())
	go backfillService.Start()
	
	log.Println("✓ Backfill service started")

	// Initialize REST API server
	restServer := rest.NewServer(config.RESTPort, db, backfillService)
	go func() {
		log.Printf("Starting REST API server on port %s", config.RESTPort)
		if err := restServer.Start(); err != nil {
			log.Printf("REST server error: %v", err)
		}
	}()

	log.Printf("✓ REST API server listening on :%s", config.RESTPort)

	// Initialize WebSocket server
	wsServer := websocket.NewServer(db, redisCache, redisPublisher)
	go func() {
		log.Printf("Starting WebSocket server on port %s", config.WSPort)
		if err := wsServer.Start(config.WSPort); err != nil {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	log.Printf("✓ WebSocket server listening on :%s", config.WSPort)
	log.Printf("✓ Minerva v%s started successfully", serviceVersion)
	log.Printf("  REST API: http://0.0.0.0:%s", config.RESTPort)
	log.Printf("  WebSocket: ws://0.0.0.0:%s", config.WSPort)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Minerva gracefully...")

	// Graceful shutdown
	cancel()
	sched.Stop() // Stop scheduler explicitly
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := restServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("REST API server shutdown error: %v", err)
	}

	time.Sleep(2 * time.Second)

	log.Println("Minerva stopped")
}

type Config struct {
	AtlasDSN    string
	RedisURL    string
	RESTPort    string
	WSPort      string
	ESPNAPIBase string
	LogLevel    string
}

func loadConfig() Config {
	return Config{
		AtlasDSN:    getEnv("ATLAS_DSN", "postgres://fortuna:fortuna_pw@localhost:5434/atlas?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		RESTPort:    getEnv("REST_PORT", "8080"),
		WSPort:      getEnv("WS_PORT", "8081"),
		ESPNAPIBase: getEnv("ESPN_API_BASE", "https://site.api.espn.com"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
