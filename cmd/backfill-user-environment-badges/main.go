// backfill-user-environment-badges は、環境バッジ(対戦環境ごとの初回対戦バッジ)機能導入前から
// 既に該当環境で対戦結果を記録していた既存ユーザーに対し、user_environment_badges へ
// 遡ってバッジを付与するための一回限りの初期投入バッチ。
//
// 判定基準は usecase.EnvironmentBadgeEvaluation.EvaluateOnMatchCreated と同じ:
// 対戦ごとに環境を求め(usecase.ResolveEnvironmentForOfficialEvent。official_event_environments
// に例外登録があればその環境、無ければ基準日時が属する環境 = environments.from_date <= 基準日時
// の中で最も新しいもの。基準日時は親recordのevent_date、無ければ対戦のcreated_at。
// usecase.RecordBasisTime参照)、ユーザー×環境の組み合わせごとに最も古い基準日時を
// achieved_at として付与する。
//
// 例外登録(official_event_environments)を後から追加・修正したときは、このバッチを再実行すると
// 付与済みのバッジを新しい判定で付け直せる。
//
// achieved_at には対戦の基準日時(event_date優先)を、created_at には達成条件に使った
// matchそのもののCreatedAtを設定する(基準日時とは別物。basisTimeは過去日を指定できて
// しまうため、created_atは実際の処理順を保つ実処理時刻寄りの値を使う)。
//
// 既に行がある組み合わせもスキップせず上書きする(判定基準の変更後に再実行して達成日時を
// 更新し直せるようにするため)。上書き対象は achieved_at / created_at のみで、record_id /
// notification_id は最初に作成された時点の値を保持する。ただし再計算しても
// achieved_at / created_at が既存値と一致する行は書き込まない(値が変わらないのに
// 全行を UPDATE しても意味が無いため)。
//
// -dry-run では、新規付与(change=create)と、値が変わる既存行(change=update。
// 変わる列を changed_fields に、現在値を current_achieved_at / current_created_at に
// 併記する)だけをログへ出し、変わらない行は completed ログの unchanged 件数にのみ
// 計上する。全件を出すと実際に変わる行が埋もれ、再実行の影響を事前に確認できなくなるため。
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
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "backfill-user-environment-badges"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず差分の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全ユーザー)")
	// ログは cmd/core-apiserver と同じJSON形式に揃える。これを呼ばないと slog の
	// 既定ハンドラ(テキスト)のままになり、usecase 層のログから layer やソース位置が落ちる。
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	flag.Parse()

	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", logging.Err(err))
	}

	db, err := postgres.NewDB(
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_USER_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		slog.Error("failed to connect database", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	environmentRepo := infrastructure.NewEnvironment(db)
	officialEventEnvironmentRepo := infrastructure.NewOfficialEventEnvironment(db)
	userEnvironmentBadgeRepo := infrastructure.NewUserEnvironmentBadge(db)

	q := db.Model(&model.User{})
	if *targetUserId != "" {
		q = q.Where("id = ?", *targetUserId)
	}

	var users []*model.User
	if tx := q.Order("id ASC").Find(&users); tx.Error != nil {
		slog.Error("failed to list users", logging.Err(tx.Error))
		os.Exit(ExitCodeNG)
	}

	// dry-run はメッセージではなく属性で出す(分岐させると grep の条件が増える)
	batchAttrs := []any{
		slog.Int("target_users", len(users)),
		slog.Bool("dry_run", *dryRun),
	}
	slog.Info("backfilling environment badges", batchAttrs...)

	total := backfillStats{}
	changedUsers := 0
	for _, user := range users {
		stats, err := backfillUser(context.Background(), db, environmentRepo, officialEventEnvironmentRepo, userEnvironmentBadgeRepo, user, *dryRun)
		// 途中で失敗しても、そこまでに処理した分は実際に反映されている(または反映予定で
		// ある)ため、集計には含める。
		total.add(stats)
		if err != nil {
			slog.Error("failed to backfill user", slog.String("user_id", user.ID), logging.Err(err))
			continue
		}
		if stats.changed() > 0 {
			changedUsers++
		}
	}

	slog.Info("completed", append(batchAttrs,
		slog.Int("changed_users", changedUsers),
		slog.Int("created", total.created),
		slog.Int("updated", total.updated),
		slog.Int("unchanged", total.unchanged),
	)...)

	os.Exit(ExitCodeOK)
}

type matchBasis struct {
	matchId  string
	recordId string
	// officialEventId は環境の例外判定(official_event_environments)に使う。
	// 環境バッジの対象は公式イベントに紐づく記録のみなので、必ず0以外になる。
	officialEventId uint
	basisTime       time.Time
	matchCreatedAt  time.Time
}

// sortBasesForAdoption は、ユーザー×環境ごとに採用する対戦を決めるための順序へ並べ替える。
// 先頭に近いものほど優先して採用される。
//
// 基準日時が同じ対戦は珍しくない(同じ記録の中の各対戦、同じ開催日の別記録)。基準日時だけで
// 比べると同点の並びが sort.Slice(非安定)とDBの返却順(ORDER BY無し)任せになり、採用される
// 対戦が実行ごとに変わる。created_at が毎回揺れて差分の確認にならないため、同点は対戦の
// 作成日時 → recordId → matchId の順で割って全順序にする。
func sortBasesForAdoption(bases []matchBasis) {
	sort.Slice(bases, func(i, j int) bool {
		if !bases[i].basisTime.Equal(bases[j].basisTime) {
			return bases[i].basisTime.Before(bases[j].basisTime)
		}
		if !bases[i].matchCreatedAt.Equal(bases[j].matchCreatedAt) {
			return bases[i].matchCreatedAt.Before(bases[j].matchCreatedAt)
		}
		if bases[i].recordId != bases[j].recordId {
			return bases[i].recordId < bases[j].recordId
		}
		return bases[i].matchId < bases[j].matchId
	})
}

// backfillStats は user_environment_badges への反映内容の内訳。
// dry-run のときは「再実行するとこうなる」という予定を表す。
type backfillStats struct {
	created   int // 行が無いため新規に付与する
	updated   int // 行はあるが achieved_at / created_at が変わるため上書きする
	unchanged int // 行があり値も変わらないため書き込まない
}

// changed は書き込みが発生する(dry-runなら発生する予定の)件数を返す。
func (s backfillStats) changed() int {
	return s.created + s.updated
}

func (s *backfillStats) add(other backfillStats) {
	s.created += other.created
	s.updated += other.updated
	s.unchanged += other.unchanged
}

// badgeChange は既存行と再計算した値を比べた結果。ログの change 属性にそのまま出す。
type badgeChange string

const (
	badgeChangeCreate badgeChange = "create"
	badgeChangeUpdate badgeChange = "update"
	badgeChangeNone   badgeChange = "none"
)

// sameStoredTime は、timestamp 列へ保存したときに同じ値になるかを返す。
//
// time.Equal(瞬間の一致)では判定できない。user_environment_badges.achieved_at は
// TIMESTAMP(without time zone)で、ドライバは Location を捨てて壁時計をそのまま格納し、
// 読み出すときに接続の TimeZone(Asia/Tokyo)を付けて返す。一方 achieved_at の元になる
// records.event_date は DATE で、UTCラベルの 00:00 として読める。そのため
// event_date(2026-08-14T00:00:00Z)を保存して読み戻すと 2026-08-14T00:00:00+09:00 になり、
// 格納されている値は同じなのに Equal は永久に false を返す。
// 格納後の値が一致するかを見たいので、Location を無視して壁時計を比べる。
func sameStoredTime(a time.Time, b time.Time) bool {
	// PostgreSQLのtimestampはマイクロ秒精度。それ未満は格納時に落ちるため比較しない。
	const layout = "2006-01-02 15:04:05.999999"

	return a.Format(layout) == b.Format(layout)
}

// classifyBadgeChange は再実行で書き込みが必要かを判定する。比較対象を achieved_at /
// created_at に限るのは、Save が conflict 時に上書きするのがこの2つだけで、
// record_id / notification_id は既存値のまま残る(＝差があっても書き込む理由にならない)ため。
func classifyBadgeChange(existing *model.UserEnvironmentBadge, achievedAt time.Time, createdAt time.Time) badgeChange {
	if existing == nil {
		return badgeChangeCreate
	}

	if sameStoredTime(existing.AchievedAt, achievedAt) && sameStoredTime(existing.CreatedAt, createdAt) {
		return badgeChangeNone
	}

	return badgeChangeUpdate
}

// changedFields は上書きで実際に変わる列を返す。achieved_at(達成日時そのもの)が動くのか、
// created_at(採用した対戦)だけが動くのかで確認の重さが違うため、ログで区別できるようにする。
func changedFields(existing *model.UserEnvironmentBadge, achievedAt time.Time, createdAt time.Time) []string {
	var fields []string
	if !sameStoredTime(existing.AchievedAt, achievedAt) {
		fields = append(fields, "achieved_at")
	}
	if !sameStoredTime(existing.CreatedAt, createdAt) {
		fields = append(fields, "created_at")
	}

	return fields
}

// backfillUser は1ユーザー分の環境バッジを補完し、その内訳を返す。
func backfillUser(
	ctx context.Context,
	db *gorm.DB,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface,
	user *model.User,
	dryRun bool,
) (backfillStats, error) {
	var matches []*model.Match
	if tx := db.Where("user_id = ?", user.ID).Find(&matches); tx.Error != nil {
		return backfillStats{}, tx.Error
	}
	if len(matches) == 0 {
		return backfillStats{}, nil
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
		return backfillStats{}, tx.Error
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
			matchId:         m.ID,
			recordId:        m.RecordId,
			officialEventId: record.OfficialEventId,
			basisTime:       usecase.RecordBasisTime(record.EventDate, record.CreatedAt),
			matchCreatedAt:  m.CreatedAt,
		})
	}
	sortBasesForAdoption(bases)

	var existing []*model.UserEnvironmentBadge
	if tx := db.Where("user_id = ?", user.ID).Find(&existing); tx.Error != nil {
		return backfillStats{}, tx.Error
	}
	existingByEnv := make(map[string]*model.UserEnvironmentBadge, len(existing))
	for _, ub := range existing {
		existingByEnv[ub.EnvironmentId] = ub
	}

	// processed は同一実行内での重複処理を防ぐためのもの(basesはbasisTime昇順なので、
	// 同じ環境について複数回対戦していても最初に到達した=最も古い基準日時を採用する)。
	// 既存データによるスキップには使わない(既存分も再計算の対象にするため)。
	processed := make(map[string]bool, len(bases))

	stats := backfillStats{}
	for _, b := range bases {
		env, err := usecase.ResolveEnvironmentForOfficialEvent(
			ctx,
			environmentRepo,
			officialEventEnvironmentRepo,
			b.officialEventId,
			b.basisTime,
		)
		if err != nil {
			if errors.Is(err, apperror.ErrRecordNotFound) {
				continue
			}
			return stats, err
		}
		if processed[env.ID] {
			continue
		}
		// 書き込むかどうかに関わらず、この環境の採用値はここで確定する。値が変わらない
		// ときに立て忘れると、同じ環境のより新しい対戦で再判定され「最も古い基準日時を
		// 採る」という前提が崩れるため、判定した時点で立てる。
		processed[env.ID] = true

		existing := existingByEnv[env.ID]
		change := classifyBadgeChange(existing, b.basisTime, b.matchCreatedAt)
		if change == badgeChangeNone {
			stats.unchanged++
			continue
		}

		attrs := []any{
			slog.String("user_id", user.ID),
			slog.String("environment_id", env.ID),
			// create=新規付与 / update=既存の付与を上書き
			slog.String("change", string(change)),
			slog.String("achieved_at", b.basisTime.Format(time.RFC3339)),
			slog.String("created_at", b.matchCreatedAt.Format(time.RFC3339)),
		}
		if change == badgeChangeUpdate {
			// 上書きで何がどう動くかは、変わる列と現在値を並べないと判断できない。
			attrs = append(attrs,
				slog.String("changed_fields", strings.Join(changedFields(existing, b.basisTime, b.matchCreatedAt), ",")),
				slog.String("current_achieved_at", existing.AchievedAt.Format(time.RFC3339)),
				slog.String("current_created_at", existing.CreatedAt.Format(time.RFC3339)),
			)
		}

		if dryRun {
			slog.Info("environment badge to backfill", append(attrs, slog.Bool("dry_run", true))...)
		} else {
			// notification_idは常に空で渡す。Save()はconflict時にachieved_at/created_atのみを
			// 上書きし、notification_idは既存値を保持する(backfill-notificationsが後から
			// 書き戻す値のため、ここで上書きしてしまわないようにするため)。
			userEnvironmentBadge := entity.NewUserEnvironmentBadge(user.ID, env.ID, b.recordId, "", b.basisTime, b.matchCreatedAt)
			if err := userEnvironmentBadgeRepo.Save(ctx, userEnvironmentBadge); err != nil {
				return stats, err
			}

			slog.Info("environment badge backfilled", append(attrs, slog.Bool("dry_run", false))...)
		}

		if change == badgeChangeCreate {
			stats.created++
		} else {
			stats.updated++
		}
	}

	return stats, nil
}
