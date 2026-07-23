package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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

// jwtSecretMinLength はVSRECORDER_JWT_SECRETに要求する最小文字数。
// HS256の鍵として妥当な強度(256bit相当)を下回る値を弾く。
const jwtSecretMinLength = 32

// validateJWTSecret はJWTの署名鍵として安全に使える値かを確認する。
//
// 未設定のまま起動すると、署名検証が空の鍵([]byte(""))で行われ、任意のuidを
// 名乗るトークンを誰でも偽造できてしまう。設定漏れは実行時に何のエラーも
// 出さないため、起動時に検知して落とす必要がある。
func validateJWTSecret(secret string) error {
	if secret == "" {
		return errors.New("VSRECORDER_JWT_SECRET is not set")
	}

	if len(secret) < jwtSecretMinLength {
		return fmt.Errorf(
			"VSRECORDER_JWT_SECRET must be at least %d characters, got %d",
			jwtSecretMinLength,
			len(secret),
		)
	}

	return nil
}

type APIServer struct {
	httpServer *http.Server
	db         *gorm.DB
}

// httpServerタイムアウト。タイムアウトを設定しない場合、ヘッダやボディを
// 少しずつ送り続ける接続(Slowloris)によってコネクションを占有され続ける。
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)

func NewAPIServer(addr string, handler http.Handler, db *gorm.DB) *APIServer {
	return &APIServer{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
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

	if err := validateJWTSecret(os.Getenv("VSRECORDER_JWT_SECRET")); err != nil {
		log.Printf("failed to validate JWT secret: %v", err)
		os.Exit(ExitCodeNG)
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
		internal.BodySizeLimitMiddleware(internal.MaxRequestBodyBytes),
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
		infrastructure.NewNotification(db),
		infrastructure.NewChampionshipSeries(db),
	)

	designationEvaluation := usecase.NewDesignationEvaluation(
		infrastructure.NewDesignation(db),
		infrastructure.NewDesignationStats(db),
		infrastructure.NewChampionshipSeries(db),
		infrastructure.NewNotification(db),
		infrastructure.NewUserPlayer(db),
	)

	environmentBadgeEvaluation := usecase.NewEnvironmentBadgeEvaluation(
		infrastructure.NewEnvironment(db),
		infrastructure.NewUserEnvironmentBadge(db),
		infrastructure.NewNotification(db),
		infrastructure.NewTransactionManager(db),
	)

	controller.NewUser(
		logger,
		r,
		infrastructure.NewUser(db),
		usecase.NewUser(
			infrastructure.NewUser(db),
			infrastructure.NewRecord(db, logger),
			infrastructure.NewDeck(db),
			infrastructure.NewDeckCode(db),
			infrastructure.NewUserPlayer(db),
			infrastructure.NewTransactionManager(db),
			badgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewTonamelEvent(
		r,
		usecase.NewTonamelEvent(
			infrastructure.NewTonamelEvent(logger),
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
		logger,
		r,
		infrastructure.NewDeck(db),
		infrastructure.NewRecord(db, logger),
		usecase.NewDeck(
			infrastructure.NewDeck(db),
			infrastructure.NewDeckAsset(logger),
			badgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewDeckCode(
		logger,
		r,
		infrastructure.NewDeckCode(db),
		infrastructure.NewRecord(db, logger),
		usecase.NewDeckCode(
			infrastructure.NewDeckCode(db),
			infrastructure.NewDeckAsset(logger),
			badgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewUserPlayer(
		logger,
		r,
		usecase.NewUserPlayer(
			infrastructure.NewUserPlayer(db),
			infrastructure.NewPokemonAvatar(db),
			infrastructure.NewPlayerRanking(db),
			infrastructure.NewTransactionManager(db),
		),
		os.Getenv("USERS_PLAYERS_LINKING_ENABLED") != "false",
	).RegisterRoute(relativePath)

	controller.NewRecord(
		r,
		infrastructure.NewRecord(db, logger),
		usecase.NewRecord(
			logger,
			infrastructure.NewRecord(db, logger),
			badgeEvaluation,
			designationEvaluation,
			infrastructure.NewTonamelEvent(logger),
			infrastructure.NewTonamelEventStore(db),
		),
	).RegisterRoute(relativePath)

	controller.NewMatch(
		r,
		infrastructure.NewMatch(db),
		infrastructure.NewRecord(db, logger),
		usecase.NewMatch(
			infrastructure.NewMatch(db),
			infrastructure.NewRecord(db, logger),
			badgeEvaluation,
			designationEvaluation,
			environmentBadgeEvaluation,
		),
	).RegisterRoute(relativePath)

	controller.NewBadge(
		r,
		usecase.NewBadge(
			infrastructure.NewBadgeDefinition(db),
			infrastructure.NewUserBadge(db),
			infrastructure.NewBadgeStats(db),
			infrastructure.NewChampionshipSeries(db),
		),
		infrastructure.NewChampionshipSeries(db),
	).RegisterRoute(relativePath)

	controller.NewEnvironmentBadge(
		r,
		usecase.NewEnvironmentBadge(
			infrastructure.NewEnvironment(db),
			infrastructure.NewUserEnvironmentBadge(db),
		),
	).RegisterRoute(relativePath)

	controller.NewNotification(
		r,
		usecase.NewNotification(
			infrastructure.NewNotification(db),
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
			infrastructure.NewChampionshipSeries(db),
			infrastructure.NewUserPlayer(db),
		),
		infrastructure.NewChampionshipSeries(db),
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

	controller.NewChampionshipSeries(
		r,
		infrastructure.NewChampionshipSeries(db),
	).RegisterRoute(relativePath)

	controller.NewUserStat(
		r,
		usecase.NewUserStat(
			infrastructure.NewUserStat(db),
			infrastructure.NewEnvironment(db),
			infrastructure.NewStandardRegulation(db),
			infrastructure.NewChampionshipSeries(db),
		),
		usecase.NewUserStatHistory(
			infrastructure.NewUserStatHistory(db),
			infrastructure.NewChampionshipSeries(db),
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
			infrastructure.NewChampionshipSeries(db),
		),
	).RegisterRoute(relativePath)

	controller.NewKizuna(
		r,
		usecase.NewKizuna(
			infrastructure.NewKizuna(db),
		),
	).RegisterRoute(relativePath)

	controller.NewOpponentDeckUsageStat(
		r,
		usecase.NewOpponentDeckUsageStat(
			infrastructure.NewOpponentDeckUsageStat(db),
			infrastructure.NewEnvironment(db),
			infrastructure.NewStandardRegulation(db),
			infrastructure.NewChampionshipSeries(db),
		),
	).RegisterRoute(relativePath)

	controller.NewOldestRecord(
		r,
		usecase.NewOldestRecord(
			infrastructure.NewOldestRecord(db),
		),
	).RegisterRoute(relativePath)

	// 活動ログのカレンダー。記録・対戦結果・デッキ・デッキコードと参照先のイベント情報を
	// まとめて返し、呼び出し側が記録1件ごとにAPIを呼ばずに済むようにする。
	controller.NewCalendar(
		r,
		usecase.NewCalendar(
			logger,
			infrastructure.NewCalendar(db),
			infrastructure.NewTonamelEventStore(db),
		),
	).RegisterRoute(relativePath)

	// プラットフォーム全体の週次デッキ使用率（公開・非会員閲覧可）。
	controller.NewWeeklyDeckUsageStat(
		r,
		usecase.NewWeeklyDeckUsageStat(
			infrastructure.NewWeeklyDeckUsageStat(db),
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
