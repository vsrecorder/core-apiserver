package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vsrecorder/core-apiserver/internal/controller"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	relativePath = "/api/v1beta"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	if _, err := config.LoadDefaultConfig(context.Background()); err != nil {
		log.Printf("failed to load default config: %v", err)
		return
	}

	dbHostname := os.Getenv("DB_HOSTNAME")
	dbPort := os.Getenv("DB_PORT")
	userName := os.Getenv("DB_USER_NAME")
	userPassword := os.Getenv("DB_USER_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	db, err := postgres.NewDB(dbHostname, dbPort, userName, userPassword, dbName)
	if err != nil {
		log.Fatalf("failed to connect database: %v\n", err)
	}

	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.Use(cors.New(cors.Config{
		AllowHeaders: []string{
			"Access-Control-Allow-Headers",
			"Access-Control-Request-Method",
			"Authorization",
			"Content-Type",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowOrigins: []string{
			"http://localhost:3000",
			"https://vsrecorder.mobi",
			"https://local.vsrecorder.mobi",
		},
		AllowCredentials: false,
		MaxAge:           24 * time.Hour,
	}))

	controller.NewUser(
		r,
		infrastructure.NewUser(db),
		usecase.NewUser(
			infrastructure.NewUser(db),
		),
	).RegisterRoute(relativePath, false)

	controller.NewTonamelEvent(
		r,
		usecase.NewTonamelEvent(
			infrastructure.NewTonamelEvent(),
		),
	).RegisterRoute(relativePath)

	controller.NewOfficialEvent(
		r,
		usecase.NewOfficialEvent(
			infrastructure.NewOfficialEvent(db),
		),
	).RegisterRoute(relativePath)

	controller.NewRecord(
		r,
		infrastructure.NewRecord(db),
		usecase.NewRecord(
			infrastructure.NewRecord(db),
		),
	).RegisterRoute(relativePath, false)

	controller.NewDeck(
		r,
		infrastructure.NewDeck(db),
		usecase.NewDeck(
			infrastructure.NewDeck(db),
		),
	).RegisterRoute(relativePath, false)

	controller.NewMatch(
		r,
		infrastructure.NewMatch(db),
		usecase.NewMatch(
			infrastructure.NewMatch(db),
		),
	).RegisterRoute(relativePath, false)

	controller.NewEnvironment(
		r,
		infrastructure.NewEnvironment(db),
	).RegisterRoute(relativePath)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    ":8914",
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 3 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
