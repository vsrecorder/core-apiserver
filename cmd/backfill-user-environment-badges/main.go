// backfill-user-environment-badges は、環境バッジ(対戦環境ごとの初回対戦バッジ)機能導入前から
// 既に該当環境で対戦結果を記録していた既存ユーザーに対し、user_environment_badges へ
// 遡ってバッジを付与するための一回限りの初期投入バッチ。
//
// 判定基準は usecase.EnvironmentBadgeEvaluation.EvaluateOnMatchCreated と同じ:
// 対戦の基準日時(親recordのevent_date、無ければ対戦のcreated_at。usecase.RecordBasisTime参照)
// が属する環境(environments.from_date <= 基準日時 の中で最も新しいもの)を対戦ごとに求め、
// ユーザー×環境の組み合わせごとに最も古い基準日時を achieved_at として付与する。
//
// achieved_at には対戦の基準日時(event_date優先)を、created_at には達成条件に使った
// matchそのもののCreatedAtを設定する(基準日時とは別物。basisTimeは過去日を指定できて
// しまうため、created_atは実際の処理順を保つ実処理時刻寄りの値を使う)。
//
// 既に行がある組み合わせもスキップせず上書きする(判定基準の変更後に再実行して達成日時を
// 更新し直せるようにするため)。上書き対象は achieved_at / created_at のみで、record_id /
// notification_id は最初に作成された時点の値を保持する。
//
// このツールは user_environment_badges 行の作成/更新のみを行い、通知(notifications)の
// 作成は行わない。バッジ獲得に対する通知の作成は backfill-notifications 側にまとめている
// (notification_id が空の行に対して通知を作成し、生成した通知IDを書き戻す)。
//
// 使い方:
//
//	# 変更内容を書き込まずに確認するだけ(デフォルト)
//	go run ./cmd/backfill-user-environment-badges
//
//	# 実際に user_environment_badges へ反映する
//	go run ./cmd/backfill-user-environment-badges -dry-run=false
//
//	# 特定ユーザーのみ対象にする(調査・検証用)
//	go run ./cmd/backfill-user-environment-badges -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず差分の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全ユーザー)")
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

	environmentRepo := infrastructure.NewEnvironment(db)
	userEnvironmentBadgeRepo := infrastructure.NewUserEnvironmentBadge(db)

	q := db.Model(&model.User{})
	if *targetUserId != "" {
		q = q.Where("id = ?", *targetUserId)
	}

	var users []*model.User
	if tx := q.Order("id ASC").Find(&users); tx.Error != nil {
		log.Printf("failed to list users: %v\n", tx.Error)
		os.Exit(ExitCodeNG)
	}

	if *dryRun {
		log.Printf("[dry-run] checking %d users (書き込みは行いません)\n", len(users))
	} else {
		log.Printf("backfilling environment badges for %d users\n", len(users))
	}

	backfilled := 0
	for _, user := range users {
		created, err := backfillUser(context.Background(), db, environmentRepo, userEnvironmentBadgeRepo, user, *dryRun)
		if err != nil {
			log.Printf("failed to backfill user=%s: %v\n", user.ID, err)
			continue
		}
		if created > 0 {
			backfilled++
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d users have environment badges to backfill\n", backfilled, len(users))
	} else {
		log.Printf("completed: backfilled %d/%d users\n", backfilled, len(users))
	}

	os.Exit(ExitCodeOK)
}

type matchBasis struct {
	recordId       string
	basisTime      time.Time
	matchCreatedAt time.Time
}

// backfillUser は1ユーザー分の環境バッジを補完する。作成した(dry-runなら作成予定の)件数を返す。
func backfillUser(
	ctx context.Context,
	db *gorm.DB,
	environmentRepo repository.EnvironmentInterface,
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface,
	user *model.User,
	dryRun bool,
) (int, error) {
	var matches []*model.Match
	if tx := db.Where("user_id = ?", user.ID).Find(&matches); tx.Error != nil {
		return 0, tx.Error
	}
	if len(matches) == 0 {
		return 0, nil
	}

	recordIdSet := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		recordIdSet[m.RecordId] = struct{}{}
	}
	recordIds := make([]string, 0, len(recordIdSet))
	for id := range recordIdSet {
		recordIds = append(recordIds, id)
	}

	var records []*model.Record
	if tx := db.Where("id IN ?", recordIds).Find(&records); tx.Error != nil {
		return 0, tx.Error
	}
	recordById := make(map[string]*model.Record, len(records))
	for _, r := range records {
		recordById[r.ID] = r
	}

	bases := make([]matchBasis, 0, len(matches))
	for _, m := range matches {
		record, ok := recordById[m.RecordId]
		if !ok {
			// 親recordが見つからない(削除済み等)場合は対象外。通常発生しない。
			continue
		}
		if record.OfficialEventId == 0 {
			// 環境バッジは公式イベントに紐づく記録のみを対象とする。
			continue
		}
		bases = append(bases, matchBasis{
			recordId:       m.RecordId,
			basisTime:      usecase.RecordBasisTime(record.EventDate, record.CreatedAt),
			matchCreatedAt: m.CreatedAt,
		})
	}
	sort.Slice(bases, func(i, j int) bool { return bases[i].basisTime.Before(bases[j].basisTime) })

	var existing []*model.UserEnvironmentBadge
	if tx := db.Where("user_id = ?", user.ID).Find(&existing); tx.Error != nil {
		return 0, tx.Error
	}
	existingByEnv := make(map[string]*model.UserEnvironmentBadge, len(existing))
	for _, ub := range existing {
		existingByEnv[ub.EnvironmentId] = ub
	}

	// processed は同一実行内での重複処理を防ぐためのもの(basesはbasisTime昇順なので、
	// 同じ環境について複数回対戦していても最初に到達した=最も古い基準日時を採用する)。
	// 既存データによるスキップには使わない(既存分も上書き対象にするため)。
	processed := make(map[string]bool, len(bases))

	created := 0
	for _, b := range bases {
		env, err := environmentRepo.FindByDate(ctx, b.basisTime)
		if err != nil {
			if errors.Is(err, apperror.ErrRecordNotFound) {
				continue
			}
			return created, err
		}
		if processed[env.ID] {
			continue
		}

		_, hasExisting := existingByEnv[env.ID]

		if dryRun {
			action := "新規付与"
			if hasExisting {
				action = "上書き"
			}
			log.Printf("[dry-run] user=%s environment=%s %s(達成日=%s)\n", user.ID, env.ID, action, b.basisTime.Format(time.RFC3339))
			processed[env.ID] = true
			created++
			continue
		}

		// notification_idは常に空で渡す。Save()はconflict時にachieved_at/created_atのみを
		// 上書きし、notification_idは既存値を保持する(backfill-notificationsが後から
		// 書き戻す値のため、ここで上書きしてしまわないようにするため)。
		userEnvironmentBadge := entity.NewUserEnvironmentBadge(user.ID, env.ID, b.recordId, "", b.basisTime, b.matchCreatedAt)
		if err := userEnvironmentBadgeRepo.Save(ctx, userEnvironmentBadge); err != nil {
			return created, err
		}

		log.Printf("user=%s environment=%s BACKFILLED achieved_at=%s\n", user.ID, env.ID, b.basisTime.Format(time.RFC3339))
		processed[env.ID] = true
		created++
	}

	return created, nil
}
