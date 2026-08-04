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
)

// UserDailyActivityCategories は既知の計測カテゴリの集合。
// 未知の値をそのまま書き込むと、集計時に誰も気づけない無音のゴミが溜まるため、
// 受け入れ判定は必ずここを通す。
var UserDailyActivityCategories = map[string]struct{}{
	UserDailyActivityCategoryVisit:  {},
	UserDailyActivityCategoryReview: {},
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
