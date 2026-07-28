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
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

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
			log.Printf("%s は %d 以上で指定してください(指定値: %d)\n", v.name, v.min, v.value)
			os.Exit(ExitCodeNG)
		}
	}

	if *minRatio <= 0 || *minRatio > 1 {
		log.Printf("-min-ratio は 0 より大きく 1 以下の割合で指定してください(60%% なら 0.6。指定値: %v)\n", *minRatio)
		os.Exit(ExitCodeNG)
	}

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

	ctx := context.Background()

	environmentRepo := infrastructure.NewEnvironment(db)
	standardRegulationRepo := infrastructure.NewStandardRegulation(db)
	championshipSeriesRepo := infrastructure.NewChampionshipSeries(db)

	// 集計期間は実行日から遡って決める(終端は当日を含めるため翌日 0 時)。
	// -supply-environment/-supply-season/-supply-regulation(需要側は demand-)が
	// 1つでも指定されていれば、週数指定より優先してそちらの期間を使う。
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	to := today.AddDate(0, 0, 1)

	supplyFrom, supplyTo, err := resolvePeriod(
		ctx, environmentRepo, standardRegulationRepo, championshipSeriesRepo,
		*supplyEnvironment, *supplySeason, *supplyRegulation,
		today.AddDate(0, 0, -7*(*supplyWeeks)), to, now,
	)
	if err != nil {
		log.Printf("failed to resolve supply period: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	demandFrom, demandTo, err := resolvePeriod(
		ctx, environmentRepo, standardRegulationRepo, championshipSeriesRepo,
		*demandEnvironment, *demandSeason, *demandRegulation,
		today.AddDate(0, 0, -7*(*demandWeeks)), to, now,
	)
	if err != nil {
		log.Printf("failed to resolve demand period: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	cfg := infrastructure.DeckNameAliasGeneratorConfig{
		SupplyFrom:      supplyFrom,
		SupplyTo:        supplyTo,
		DemandFrom:      demandFrom,
		DemandTo:        demandTo,
		MinSupport:      *minSupport,
		MinRatio:        *minRatio,
		MinContributors: *minContributors,
		MinAliasRunes:   *minAliasRunes,
	}

	log.Printf(
		"教師データ %s〜%s / 救済対象 %s〜%s / しきい値: 支持%d件以上・占有率%.0f%%以上・%d人以上・%d文字以上\n",
		cfg.SupplyFrom.Format("2006-01-02"),
		formatInclusiveTo(cfg.SupplyTo),
		cfg.DemandFrom.Format("2006-01-02"),
		formatInclusiveTo(cfg.DemandTo),
		cfg.MinSupport,
		cfg.MinRatio*100,
		cfg.MinContributors,
		cfg.MinAliasRunes,
	)

	candidates, rejected, err := infrastructure.GenerateDeckNameAliasCandidates(ctx, db, cfg)
	if err != nil {
		log.Printf("failed to generate deck name alias candidates: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	rescuedVotes := 0
	for _, c := range candidates {
		rescuedVotes += c.DemandVotes
		log.Printf(
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

	log.Printf("候補 %d 件 / 救済見込み %d 票\n", len(candidates), rescuedVotes)

	if *showRejected {
		printRejected(rejected, *rejectedLimit)
	}

	if *dryRun {
		log.Printf("[dry-run] 書き込みは行いません(-dry-run=false で反映)\n")
		os.Exit(ExitCodeOK)
	}

	saved, err := infrastructure.ReplaceAutoDeckNameAliases(ctx, db, candidates)
	if err != nil {
		log.Printf("failed to replace auto deck name aliases: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	log.Printf("完了: source='auto' を %d 行で再生成しました\n", saved)
	os.Exit(ExitCodeOK)
}

// resolvePeriod は environmentId/season/regulationId のいずれかが指定されていればその期間
// (期間が複数指定された場合は交差)を、いずれも空文字なら weeksFrom〜weeksTo(週数指定による
// 既定の期間)をそのまま返す。
func resolvePeriod(
	ctx context.Context,
	environmentRepo repository.EnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	environmentId string,
	season string,
	regulationId string,
	weeksFrom time.Time,
	weeksTo time.Time,
	now time.Time,
) (time.Time, time.Time, error) {
	if environmentId == "" && season == "" && regulationId == "" {
		return weeksFrom, weeksTo, nil
	}

	return usecase.PeriodDateRange(
		ctx, environmentRepo, standardRegulationRepo, championshipSeriesRepo,
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
func printRejected(rejected []*infrastructure.DeckNameAliasRejection, limit int) {
	log.Printf("--- 候補にならなかったデッキ名 %d 件(救済し損ねた票の多い順) ---\n", len(rejected))

	for i, r := range rejected {
		if limit > 0 && i >= limit {
			log.Printf("  ...ほか %d 件(-rejected-limit=0 で全件表示)\n", len(rejected)-limit)
			break
		}

		// 教師データがある落選理由(支持・占有率・人数)だけ診断値を添える。
		if r.TotalSupply > 0 {
			log.Printf(
				"  %-24s 逃し%4d票  理由:%-14s (支持%d/%d件 %.0f%% %d人)\n",
				r.Alias, r.DemandVotes, rejectReasonLabel(r.Reason),
				r.Support, r.TotalSupply, r.Ratio*100, r.Contributors,
			)
		} else {
			log.Printf(
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
