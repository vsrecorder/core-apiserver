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
)

// UserDailyActivityCategories は既知の計測カテゴリの集合。
// 未知の値をそのまま書き込むと、集計時に誰も気づけない無音のゴミが溜まるため、
// 受け入れ判定は必ずここを通す。
var UserDailyActivityCategories = map[string]struct{}{
	UserDailyActivityCategoryVisit:       {},
	UserDailyActivityCategoryReview:      {},
	UserDailyActivityCategoryStandalone:  {},
	UserDailyActivityCategoryPushCapable: {},
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
