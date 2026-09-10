// generate-deck-name-aliases は、デッキ名エイリアス辞書(deck_name_aliases)を
// 実データの共起から自動生成するバッチ。
//
// 週次デッキ使用率の集計では、スプライト未設定の票をデッキ名から推測して救済している
// (internal/infrastructure/deck_name.go)。その辞書を人手で育てる代わりに、
// 「デッキ名とスプライトを両方登録している記録」を教師データとして代表構成を求め、
// 現在スプライト未設定で除外されている名前ぶんだけエイリアスを作る。
//
// 冪等性: source='auto' の行だけを毎回全削除→再生成する。
// 人が登録した source='manual' の行は読むだけで書き換えない。
// 手動エントリで既に解決できる名前は候補にしない(手動の意図を尊重する)。
//
// 使い方:
//
//	# 生成される候補を確認するだけ(デフォルト。DBは変更しない)
//	go run ./cmd/generate-deck-name-aliases
//
//	# 実際に deck_name_aliases へ反映する
//	go run ./cmd/generate-deck-name-aliases -dry-run=false
//
//	# しきい値と集計期間(週数指定)を調整する
//	go run ./cmd/generate-deck-name-aliases -min-support=20 -min-ratio=0.7 -supply-weeks=24
//
//	# 教師データの抽出対象期間・救済対象期間を環境/シーズン/レギュレーションで指定する
//	# (指定した場合はそれぞれ -supply-weeks / -demand-weeks より優先される。複数指定時は期間の交差を取る)
//	go run ./cmd/generate-deck-name-aliases -supply-season=2026 -demand-environment=sv9a
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "generate-deck-name-aliases"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	// ログは cmd/core-apiserver と同じJSON形式に揃える。これを呼ばないと slog の
	// 既定ハンドラ(テキスト)のままになり、usecase 層のログから layer やソース位置が落ちる。
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	defaults := infrastructure.DefaultDeckNameAliasGeneratorConfig()

	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず生成される候補の確認のみ行う")
	supplyWeeks := flag.Int("supply-weeks", 12, "教師データ(名前とスプライトが両方ある記録)を遡る週数(-supply-environment/-supply-season/-supply-regulation未指定時のみ使う)")
	demandWeeks := flag.Int("demand-weeks", 4, "救済対象(スプライト未設定の票)を遡る週数(-demand-environment/-demand-season/-demand-regulation未指定時のみ使う)")
	supplyEnvironment := flag.String("supply-environment", "", "教師データの抽出対象期間を環境ID(environments.id)で指定する。season/regulationと併用時は期間の交差を取る")
	supplySeason := flag.String("supply-season", "", "教師データの抽出対象期間をシーズン(championship_series.idから接頭辞series_を除いた識別子。例:2026)で指定する")
	supplyRegulation := flag.String("supply-regulation", "", "教師データの抽出対象期間をレギュレーションID(standard_regulations.id)で指定する")
	demandEnvironment := flag.String("demand-environment", "", "救済対象期間を環境ID(environments.id)で指定する。season/regulationと併用時は期間の交差を取る")
	demandSeason := flag.String("demand-season", "", "救済対象期間をシーズン(championship_series.idから接頭辞series_を除いた識別子。例:2026)で指定する")
	demandRegulation := flag.String("demand-regulation", "", "救済対象期間をレギュレーションID(standard_regulations.id)で指定する")
	minSupport := flag.Int("min-support", defaults.MinSupport, "代表構成の支持件数の下限")
	minRatio := flag.Float64("min-ratio", defaults.MinRatio, "代表構成の占有率の下限。割合で指定する(60% なら 0.6)")
	minContributors := flag.Int("min-contributors", defaults.MinContributors, "代表構成を使った実ユーザー数の下限")
	minAliasRunes := flag.Int("min-alias-runes", defaults.MinAliasRunes, "生成するエイリアスの最小文字数")
	showRejected := flag.Bool("show-rejected", false, "候補にならなかったデッキ名も理由つきで表示する")
	rejectedLimit := flag.Int("rejected-limit", 30, "-show-rejected 時に表示する最大件数(救済見込み票の多い順。0 で全件)")
	flag.Parse()

	// しきい値の指定ミスは「候補0件」という正常終了に紛れて気づけないため、ここで弾く。
	// 特に占有率はログに % で出す一方で指定は割合(0.6)のため、60 と書かれやすい。
	for _, v := range []struct {
		name  string
		value int
		min   int
	}{
		{"-supply-weeks", *supplyWeeks, 1},
		{"-demand-weeks", *demandWeeks, 1},
		{"-min-support", *minSupport, 0},
		{"-min-contributors", *minContributors, 0},
		{"-min-alias-runes", *minAliasRunes, 1},
		{"-rejected-limit", *rejectedLimit, 0},
	} {
		if v.value < v.min {
			slog.Error("flag value is too small",
				slog.String("flag", v.name), slog.Int("min", v.min), slog.Int("value", v.value))
			os.Exit(ExitCodeNG)
		}
	}

	if *minRatio <= 0 || *minRatio > 1 {
		// 60% なら 0.6 のように、0 より大きく 1 以下の割合で指定する
		slog.Error("-min-ratio must be in (0, 1]", slog.Float64("value", *minRatio))
		os.Exit(ExitCodeNG)
	}

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

	ctx := context.Background()

	environmentRepo := infrastructure.NewEnvironment(db)
	officialEventEnvironmentRepo := infrastructure.NewOfficialEventEnvironment(db)
	standardRegulationRepo := infrastructure.NewStandardRegulation(db)
	championshipSeriesRepo := infrastructure.NewChampionshipSeries(db)

	// 集計期間は実行日から遡って決める(終端は当日を含めるため翌日 0 時)。
	// -supply-environment/-supply-season/-supply-regulation(需要側は demand-)が
	// 1つでも指定されていれば、週数指定より優先してそちらの期間を使う。
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	to := today.AddDate(0, 0, 1)

	supplyPeriod, err := resolvePeriod(
		ctx, environmentRepo, officialEventEnvironmentRepo, standardRegulationRepo, championshipSeriesRepo,
		*supplyEnvironment, *supplySeason, *supplyRegulation,
		today.AddDate(0, 0, -7*(*supplyWeeks)), to, now,
	)
	if err != nil {
		slog.Error("failed to resolve supply period", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	demandPeriod, err := resolvePeriod(
		ctx, environmentRepo, officialEventEnvironmentRepo, standardRegulationRepo, championshipSeriesRepo,
		*demandEnvironment, *demandSeason, *demandRegulation,
		today.AddDate(0, 0, -7*(*demandWeeks)), to, now,
	)
	if err != nil {
		slog.Error("failed to resolve demand period", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	cfg := infrastructure.DeckNameAliasGeneratorConfig{
		SupplyPeriod:    supplyPeriod,
		DemandPeriod:    demandPeriod,
		MinSupport:      *minSupport,
		MinRatio:        *minRatio,
		MinContributors: *minContributors,
		MinAliasRunes:   *minAliasRunes,
	}

	// 実行条件はログに残す(後からどの設定で作った辞書かを追えるようにするため)。
	// 候補・落選の一覧そのものは人が読んで判断する表なので、ログ(JSON・stderr)ではなく
	// 標準出力へ出す。JSONに混ぜると桁が揃わず、目視での比較ができなくなる。
	slog.Info("generating deck name aliases",
		slog.String("supply_from", cfg.SupplyPeriod.From.Format("2006-01-02")),
		slog.String("supply_to", formatInclusiveTo(cfg.SupplyPeriod.To)),
		slog.String("demand_from", cfg.DemandPeriod.From.Format("2006-01-02")),
		slog.String("demand_to", formatInclusiveTo(cfg.DemandPeriod.To)),
		slog.Int("min_support", cfg.MinSupport),
		slog.Float64("min_ratio", cfg.MinRatio),
		slog.Int("min_contributors", cfg.MinContributors),
		slog.Int("min_alias_runes", cfg.MinAliasRunes),
	)

	candidates, rejected, err := infrastructure.GenerateDeckNameAliasCandidates(ctx, db, cfg)
	if err != nil {
		slog.Error("failed to generate deck name alias candidates", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	rescuedVotes := 0
	for _, c := range candidates {
		rescuedVotes += c.DemandVotes
		fmt.Printf(
			"  %-24s → %-32s 救済%4d票 (支持%d/%d件 %.0f%% %d人)\n",
			c.Alias,
			formatSprites(c.Sprites),
			c.DemandVotes,
			c.Support,
			c.TotalSupply,
			c.Ratio*100,
			c.Contributors,
		)
	}

	slog.Info("generated candidates",
		slog.Int("candidates", len(candidates)), slog.Int("rescued_votes", rescuedVotes))

	if *showRejected {
		printRejected(rejected, *rejectedLimit)
	}

	if *dryRun {
		slog.Info("skipped writing: -dry-run=false applies the candidates", slog.Bool("dry_run", true))
		os.Exit(ExitCodeOK)
	}

	saved, err := infrastructure.ReplaceAutoDeckNameAliases(ctx, db, candidates)
	if err != nil {
		slog.Error("failed to replace auto deck name aliases", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	slog.Info("completed: regenerated aliases with source='auto'", slog.Int64("saved", int64(saved)))
	os.Exit(ExitCodeOK)
}

// resolvePeriod は environmentId/season/regulationId のいずれかが指定されていればその期間
// (期間が複数指定された場合は交差)を、いずれも空文字なら weeksFrom〜weeksTo(週数指定による
// 既定の期間)をそのまま返す。
//
// 環境を指定した場合の戻り値には、開催日と実際の対戦環境がズレる公式イベントの例外も
// 含まれる(usecase.StatPeriodFor 参照)。
func resolvePeriod(
	ctx context.Context,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	environmentId string,
	season string,
	regulationId string,
	weeksFrom time.Time,
	weeksTo time.Time,
	now time.Time,
) (repository.StatPeriod, error) {
	if environmentId == "" && season == "" && regulationId == "" {
		return repository.StatPeriod{
			From:     weeksFrom,
			To:       weeksTo,
			BaseFrom: weeksFrom,
			BaseTo:   weeksTo,
		}, nil
	}

	return usecase.StatPeriodFor(
		ctx, environmentRepo, officialEventEnvironmentRepo, standardRegulationRepo, championshipSeriesRepo,
		environmentId, season, regulationId, now,
	)
}

// formatInclusiveTo は集計期間の終端をログ用に整形する。
// SupplyTo/DemandTo は半開区間 [from, to) の排他的上限(集計は event_date < to)のため、
// そのまま出すと集計対象に含まれない日付が期間の末日に見えてしまう。前日を末日として表示する。
func formatInclusiveTo(to time.Time) string {
	return to.AddDate(0, 0, -1).Format("2006-01-02")
}

// formatSprites はログ用に代表スプライトを "0006(1) 0018(2)" 形式へ整形する。
func formatSprites(sprites []infrastructure.DeckNameAliasSprite) string {
	parts := make([]string, 0, len(sprites))
	for _, s := range sprites {
		parts = append(parts, fmt.Sprintf("%s(%d)", s.PokemonSpriteId, s.Position))
	}

	return strings.Join(parts, " ")
}

// printRejected は候補にならなかったデッキ名を、救済見込み票の多い順に理由つきで表示する。
// limit が 0 のときは全件表示する。
//
// 桁を揃えて目視で比較する表のため、ログ(JSON・stderr)ではなく標準出力へ出す。
func printRejected(rejected []*infrastructure.DeckNameAliasRejection, limit int) {
	fmt.Printf("--- 候補にならなかったデッキ名 %d 件(救済し損ねた票の多い順) ---\n", len(rejected))

	for i, r := range rejected {
		if limit > 0 && i >= limit {
			fmt.Printf("  ...ほか %d 件(-rejected-limit=0 で全件表示)\n", len(rejected)-limit)
			break
		}

		// 教師データがある落選理由(支持・占有率・人数)だけ診断値を添える。
		if r.TotalSupply > 0 {
			fmt.Printf(
				"  %-24s 逃し%4d票  理由:%-14s (支持%d/%d件 %.0f%% %d人)\n",
				r.Alias, r.DemandVotes, rejectReasonLabel(r.Reason),
				r.Support, r.TotalSupply, r.Ratio*100, r.Contributors,
			)
		} else {
			fmt.Printf(
				"  %-24s 逃し%4d票  理由:%s\n",
				r.Alias, r.DemandVotes, rejectReasonLabel(r.Reason),
			)
		}
	}
}

// rejectReasonLabel は落選理由コードを日本語ラベルにする。
func rejectReasonLabel(reason string) string {
	switch reason {
	case infrastructure.DeckNameAliasRejectTooShort:
		return "短すぎる"
	case infrastructure.DeckNameAliasRejectManualExists:
		return "手動辞書で解決済"
	case infrastructure.DeckNameAliasRejectNoSupply:
		return "教師データなし"
	case infrastructure.DeckNameAliasRejectLowSupport:
		return "支持不足"
	case infrastructure.DeckNameAliasRejectLowRatio:
		return "占有率不足"
	case infrastructure.DeckNameAliasRejectFewContributors:
		return "人数不足"
	default:
		return reason
	}
}
