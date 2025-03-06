package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebaseV4 "firebase.google.com/go/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vsrecorder/core-apiserver/internal/controller"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/firebase"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"google.golang.org/api/option"
)

const (
	relativePath = "/api/v1beta"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
		return
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

	firebaseProjectId := os.Getenv("FIREBASE_PROJECT_ID")
	opt := option.WithCredentialsJSON([]byte(os.Getenv("FIREBASE_CREDENTIALS_JSON")))
	config := &firebaseV4.Config{ProjectID: firebaseProjectId}
	authClient, err := firebase.NewClient(config, opt)
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
		usecase.NewUser(
			infrastructure.NewUser(authClient),
		),
	).RegisterRoute(relativePath)

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

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 3 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 3 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	// catching ctx.Done(). timeout of 3 seconds.
	<-ctx.Done()
	log.Println("timeout of 3 seconds.")

	log.Println("Server exiting")
}
