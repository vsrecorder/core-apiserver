// cleanup-stale-streak-notifications は、連続記録が途切れているのに取り消されず残って
// しまった「ストリークを継続中です」通知を一括で掃除する復旧バッチ。
//
// この通知は「今もN週続いている」という現在進行形の状態を伝えるものだが、記録の作成・
// 削除・更新で取り消す仕組み(usecase.BadgeEvaluation.RevokeStaleStreakNotifications)を
// 入れる前に作られた通知は残ったままになる。本ツールはその取り消しロジックをそのまま
// 呼ぶので、実行時点の連続週数では成立しない通知だけが消える(今も成立している週数の
// 通知は残る)。
//
// 仕組みが入って以降に必要になるものではないため、定期実行はしない(記録をさわらないまま
// 時間が経って途切れたぶんは、あえて消しに行かず次の書き込み時の再計算に任せる方針)。
//
// 使い方:
//
//	# 削除は行わず対象を確認するだけ(デフォルト)
//	go run ./cmd/cleanup-stale-streak-notifications
//
//	# 実際に削除する
//	go run ./cmd/cleanup-stale-streak-notifications -dry-run=false
//
//	# 特定ユーザーのみ対象にする(調査・検証用)
//	go run ./cmd/cleanup-stale-streak-notifications -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、削除は行わず対象の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定ならストリーク通知を持つ全ユーザー)")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	db, err := postgres.NewDB(
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_USER_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		log.Printf("failed to connect database: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	badgeEvaluation := usecase.NewBadgeEvaluation(
		&cachedBadgeDefinition{inner: infrastructure.NewBadgeDefinition(db)},
		infrastructure.NewUserBadge(db),
		infrastructure.NewUserStreak(db),
		infrastructure.NewBadgeStats(db),
		infrastructure.NewNotification(db),
		infrastructure.NewChampionshipSeries(db),
		infrastructure.NewTransactionManager(db),
	)

	ctx := context.Background()

	var userIds []string
	if *targetUserId != "" {
		userIds = []string{*targetUserId}
	} else {
		userIds, err = findTargetUserIds(db)
		if err != nil {
			log.Printf("failed to list users: %v\n", err)
			os.Exit(ExitCodeNG)
		}
	}

	if *dryRun {
		log.Printf("[dry-run] checking stale streak notifications for %d users (削除は行いません)\n", len(userIds))
	} else {
		log.Printf("cleaning up stale streak notifications for %d users\n", len(userIds))
	}

	affectedUsers := 0
	staleNotifications := 0

	for _, userId := range userIds {
		stale, err := badgeEvaluation.RevokeStaleStreakNotifications(ctx, userId, *dryRun)
		if err != nil {
			log.Printf("failed to revoke stale streak notifications user=%s: %v\n", userId, err)
			continue
		}
		if len(stale) == 0 {
			continue
		}

		affectedUsers++
		staleNotifications += len(stale)

		for _, n := range stale {
			if *dryRun {
				log.Printf("[dry-run] STALE user=%s id=%s created_at=%s body=%s\n", userId, n.ID, n.CreatedAt.Format("2006-01-02 15:04:05"), n.Body)
			} else {
				log.Printf("DELETED user=%s id=%s created_at=%s body=%s\n", userId, n.ID, n.CreatedAt.Format("2006-01-02 15:04:05"), n.Body)
			}
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d stale notifications on %d/%d users\n", staleNotifications, affectedUsers, len(userIds))
	} else {
		log.Printf("completed: deleted %d notifications on %d/%d users\n", staleNotifications, affectedUsers, len(userIds))
	}

	os.Exit(ExitCodeOK)
}

// cachedBadgeDefinition はバッジ定義の取得を1回だけに抑えるラッパー。
// 取り消し判定はユーザーごとに定義一覧を引くため、素通しだと対象ユーザー数だけ
// 同じクエリを繰り返してしまう。定義はマスタデータで、バッチ実行中に変わらない前提。
type cachedBadgeDefinition struct {
	inner repository.BadgeDefinitionInterface
	cache []*entity.BadgeDefinition
}

func (c *cachedBadgeDefinition) FindAll(ctx context.Context) ([]*entity.BadgeDefinition, error) {
	if c.cache != nil {
		return c.cache, nil
	}

	definitions, err := c.inner.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	c.cache = definitions

	return definitions, nil
}

// findTargetUserIds はストリークカテゴリの通知を1件でも持つユーザーを返す。
// 取り消し対象になりうるのはこの通知を持つユーザーだけなので、全ユーザーを
// 走査せずここまで絞り込む(同じカテゴリの途切れ防止nudgeしか持たないユーザーも
// 含まれるが、本文の完全一致で判定するため誤って削除されることはない)。
func findTargetUserIds(db *gorm.DB) ([]string, error) {
	var userIds []string

	if tx := db.Table("notifications").
		Where("category = ?", usecase.NotificationCategoryStreak).
		Distinct("user_id").
		Pluck("user_id", &userIds); tx.Error != nil {
		return nil, tx.Error
	}

	return userIds, nil
}
