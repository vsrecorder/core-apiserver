package entity

import (
	"time"
)

// 「見る」利用の計測カテゴリ (USER_DAILY_ACTIVITIES_PLAN.md §3)。
//
// 新しいカテゴリを追加するときは、定数を1つ足して UserDailyActivityCategories に
// 登録するだけでよい。DBスキーマの変更は不要
// (user_daily_activities.category は CHECK 制約を持たない VARCHAR のため)。
// 一度使ったカテゴリ名は意味を変えない・使い回さない(過去データの解釈が壊れるため)。
const (
	// UserDailyActivityCategoryVisit はその日サイトを開いたシグナル(全ページ共通)。
	UserDailyActivityCategoryVisit = "visit"

	// UserDailyActivityCategoryReview はその日自分の戦績を見返したシグナル。
	// どのページを見返しとみなすかの判断はクライアント側が持つ。
	UserDailyActivityCategoryReview = "review"

	// UserDailyActivityCategoryStandalone はその日ホーム画面に追加したPWAから
	// 開いたシグナル(display-mode: standalone)。visit の部分集合。
	// Web Push(B-1)の投資判断に必要な「PWA起動比率」を測るために追加した
	// (WAU_RECOVERY_EXECUTION_PLAN.md Step 0-C)。
	UserDailyActivityCategoryStandalone = "standalone"

	// UserDailyActivityCategoryPushCapable はその日の起動環境で Web Push API が
	// 利用可能だったシグナル(ServiceWorker/PushManager/Notification が揃っている)。
	// 許諾の可否ではなく API の有無であり、これがそのまま B-1 の到達率の上限になる。
	// iOS はホーム画面追加したPWAでしか PushManager が生えないため、
	// standalone との差がそのまま「iOSでインストールされていない層」の規模を表す。
	UserDailyActivityCategoryPushCapable = "push_capable"

	// UserDailyActivityCategoryReport はその日バトルレポート(/users/report 配下)を開いた
	// シグナル。review の部分集合。週次レポート通知(P-2)の閲覧率を、通知した人のうち
	// 何人がレポートまで辿り着いたかで測るために分けて持つ(P2_WEEKLY_REPORT_PLAN.md)。
	UserDailyActivityCategoryReport = "report"

	// 以下3つは「登録してから初回記録に辿り着くまで、どのステップで落ちたか」を測る
	// (A系の再設計の前提・engagement-weekly-2026-09-14.md §5.6)。
	//
	// W0記録率は7コホート連続で50%前後から動かないのに、A-1〜A-3 はすべて実装済みで
	// 手が尽きているように見える。これは施策が効かないのではなく、
	// 「記録したか / しなかったか」の2値しか測っておらず、どこで落ちているかを
	// 一度も観測していないため。GAイベント(cta_first_record_impression 等)は既に
	// 飛んでいるが、Grafana からは読めずコホート分析にも繋げられない。
	//
	// 登録 → visit → onboarding_cta → record_form → records の5段で見れば、
	// 「ホームに来ていない」「CTAを見ていない」「フォームで諦めた」のどれかに絞れる。

	// UserDailyActivityCategoryOnboardingCta はその日、記録0件の空状態CTA
	// (FirstRecordCtaCard / QuickStartModal)が表示されたシグナル。
	// 表示条件が「記録0件」なので、初回記録後は二度と立たない。
	UserDailyActivityCategoryOnboardingCta = "onboarding_cta"

	// UserDailyActivityCategoryRecordForm はその日、記録作成フォーム
	// (/records/quick・/records/create)を開いたシグナル。
	// 記録経験者も日常的に立てるため、初回記録ファネルとして読むときは
	// 「初回記録より前に立った分」だけを見ること。
	UserDailyActivityCategoryRecordForm = "record_form"

	// UserDailyActivityCategoryDeckForm はその日、デッキ登録フォームを開いたシグナル。
	// A-2(デッキコード→記録フォーム直行)の副導線がどこまで進んだかを測る。
	UserDailyActivityCategoryDeckForm = "deck_form"
)

// UserDailyActivityCategories は既知の計測カテゴリの集合。
// 未知の値をそのまま書き込むと、集計時に誰も気づけない無音のゴミが溜まるため、
// 受け入れ判定は必ずここを通す。
var UserDailyActivityCategories = map[string]struct{}{
	UserDailyActivityCategoryVisit:         {},
	UserDailyActivityCategoryReview:        {},
	UserDailyActivityCategoryStandalone:    {},
	UserDailyActivityCategoryPushCapable:   {},
	UserDailyActivityCategoryReport:        {},
	UserDailyActivityCategoryOnboardingCta: {},
	UserDailyActivityCategoryRecordForm:    {},
	UserDailyActivityCategoryDeckForm:      {},
}

// IsKnownUserDailyActivityCategory は既知のカテゴリかどうかを返す。
func IsKnownUserDailyActivityCategory(category string) bool {
	_, ok := UserDailyActivityCategories[category]

	return ok
}

type UserDailyActivity struct {
	UserId string
	// Date はJST基準の日付(アプリを開いた日)。クライアントの時計を信用せず
	// サーバ側で当日に確定させる。
	Date      time.Time
	Category  string
	UpdatedAt time.Time
}

func NewUserDailyActivity(
	userId string,
	date time.Time,
	category string,
	updatedAt time.Time,
) *UserDailyActivity {
	return &UserDailyActivity{
		UserId:    userId,
		Date:      date,
		Category:  category,
		UpdatedAt: updatedAt,
	}
}
