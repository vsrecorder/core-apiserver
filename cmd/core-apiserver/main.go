package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vsrecorder/core-apiserver/internal"
	"github.com/vsrecorder/core-apiserver/internal/controller"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

const (
	relativePath = "/api/v1beta"
	appName      = "core-apiserver"
)

type APIServer struct {
	httpServer *http.Server
	db         *gorm.DB
}

func NewAPIServer(addr string, handler http.Handler, db *gorm.DB) *APIServer {
	return &APIServer{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		db: db,
	}
}

func (s *APIServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}

	errCh := make(chan error, 1)

	go func() {
		if err := s.httpServer.Serve(ln); err != nil &&
			err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server error: %w", err)

	case <-ctx.Done():
		return s.Shutdown()
	}
}

func (s *APIServer) Shutdown() error {
	log.Println("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
		return err
	}

	log.Println("cleanup: closing DB connection...")

	if sqlDB, err := s.db.DB(); err != nil {
		log.Printf("db close error: %v", err)
	} else {
		if err := sqlDB.Close(); err != nil {
			log.Printf("db close error: %v", err)
		}
	}

	log.Println("server exited cleanly")

	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	if _, err := config.LoadDefaultConfig(context.Background()); err != nil {
		log.Printf("failed to load default config: %v", err)
		os.Exit(ExitCodeNG)
	}

	dbHostname := os.Getenv("DB_HOSTNAME")
	dbPort := os.Getenv("DB_PORT")
	userName := os.Getenv("DB_USER_NAME")
	userPassword := os.Getenv("DB_USER_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	db, err := postgres.NewDB(dbHostname, dbPort, userName, userPassword, dbName)
	if err != nil {
		log.Printf("failed to connect database: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	logger := internal.InitLogger(internal.LogConfig{
		Level:   "info",
		AppName: appName,
	})

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(
		internal.RequestIDMiddleware(),
		internal.AccessLogMiddleware(logger),
		gin.Recovery(),
	)

	r.SetTrustedProxies(nil)
	r.Use(cors.New(cors.Config{
		AllowHeaders: []string{
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
		MaxAge:           1 * time.Hour,
	}))

	badgeEvaluation := usecase.NewBadgeEvaluation(
		infrastructure.NewBadgeDefinition(db),
		infrastructure.NewUserBadge(db),
		infrastructure.NewUserStreak(db),
		infrastructure.NewBadgeStats(db),
	)

	controller.NewUser(
		logger,
		r,
		infrastructure.NewUser(db),
		usecase.NewUser(
			infrastructure.NewUser(db),
			infrastructure.NewRecord(db, slog.Default()),
			infrastructure.NewDeck(db),
			infrastructure.NewDeckCode(db),
			infrastructure.NewTransactionManager(db),
			badgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewTonamelEvent(
		r,
		usecase.NewTonamelEvent(
			infrastructure.NewTonamelEvent(slog.Default()),
		),
	).RegisterRoute(relativePath)

	controller.NewOfficialEvent(
		r,
		usecase.NewOfficialEvent(
			infrastructure.NewOfficialEvent(db),
		),
	).RegisterRoute(relativePath)

	controller.NewUnofficialEvent(
		r,
		usecase.NewUnofficialEvent(
			infrastructure.NewUnofficialEvent(db),
		),
	).RegisterRoute(relativePath)

	controller.NewDeck(
		r,
		infrastructure.NewDeck(db),
		infrastructure.NewRecord(db, slog.Default()),
		usecase.NewDeck(
			infrastructure.NewDeck(db),
			badgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewDeckCode(
		r,
		infrastructure.NewDeckCode(db),
		infrastructure.NewRecord(db, slog.Default()),
		usecase.NewDeckCode(
			infrastructure.NewDeckCode(db),
		),
	).RegisterRoute(relativePath)

	controller.NewRecord(
		r,
		infrastructure.NewRecord(db, slog.Default()),
		usecase.NewRecord(
			infrastructure.NewRecord(db, slog.Default()),
			badgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewMatch(
		r,
		infrastructure.NewMatch(db),
		infrastructure.NewRecord(db, slog.Default()),
		usecase.NewMatch(
			infrastructure.NewMatch(db),
			badgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewBadge(
		r,
		usecase.NewBadge(
			infrastructure.NewBadgeDefinition(db),
			infrastructure.NewUserBadge(db),
			infrastructure.NewBadgeStats(db),
		),
	).RegisterRoute(relativePath)

	controller.NewStreak(
		r,
		usecase.NewStreak(
			infrastructure.NewUserStreak(db),
		),
	).RegisterRoute(relativePath)

	controller.NewDesignation(
		r,
		usecase.NewDesignation(
			infrastructure.NewDesignation(db),
			infrastructure.NewDesignationStats(db),
		),
	).RegisterRoute(relativePath)

	controller.NewEnvironment(
		r,
		infrastructure.NewEnvironment(db),
	).RegisterRoute(relativePath)

	controller.NewCityleagueSchedule(
		r,
		infrastructure.NewCityleagueSchedule(db),
	).RegisterRoute(relativePath)

	controller.NewCityleagueResult(
		r,
		infrastructure.NewCityleagueResult(db),
	).RegisterRoute(relativePath)

	controller.NewStandardRegulation(
		r,
		infrastructure.NewStandardRegulation(db),
	).RegisterRoute(relativePath)

	controller.NewUserStat(
		r,
		usecase.NewUserStat(
			infrastructure.NewUserStat(db),
			infrastructure.NewEnvironment(db),
			infrastructure.NewStandardRegulation(db),
		),
		usecase.NewUserStatHistory(
			infrastructure.NewUserStatHistory(db),
		),
		usecase.NewUserStatRecent(
			infrastructure.NewUserStatRecent(db),
			infrastructure.NewEnvironment(db),
		),
	).RegisterRoute(relativePath)

	controller.NewDeckUsageStat(
		r,
		usecase.NewDeckUsageStat(
			infrastructure.NewDeckUsageStat(db),
			infrastructure.NewEnvironment(db),
			infrastructure.NewStandardRegulation(db),
		),
	).RegisterRoute(relativePath)

	controller.NewOpponentDeckUsageStat(
		r,
		usecase.NewOpponentDeckUsageStat(
			infrastructure.NewOpponentDeckUsageStat(db),
			infrastructure.NewEnvironment(db),
			infrastructure.NewStandardRegulation(db),
		),
	).RegisterRoute(relativePath)

	{
		ctx, stop := signal.NotifyContext(
			context.Background(),
			syscall.SIGINT,
			syscall.SIGTERM,
		)
		defer stop()

		server := NewAPIServer(":8914", r, db)

		if err := server.Start(ctx); err != nil {
			log.Printf("server error: %v", err)
			os.Exit(ExitCodeNG)
		}

		os.Exit(ExitCodeOK)
	}
}
