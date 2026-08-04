package dto

// UserDailyActivityRequest は「見る」利用の計測ビーコンのリクエスト。
//
// categories を配列で受けるのは、カテゴリが増えてもリクエスト数を増やさないため。
// 有効なカテゴリの列挙はここに書かない(既知かどうかの判定は
// entity.UserDailyActivityCategories に一元化する。→ USER_DAILY_ACTIVITIES_PLAN.md §3.3)。
type UserDailyActivityRequest struct {
	Categories []string `json:"categories"`
}
