package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// 反応の無い人への push を間引くためのガードレール(B1_B2_PUSH_NOTIFICATION_PLAN.md §5.5)。
// 反応しない人に毎週撃つのは許諾取り消しの最短経路で、取り消しは回復不能。
const (
	// pushQuietAfterUnclicked は「反応が無い」とみなす直近の配信回数。
	// この回数連続で未タップ、かつその間に記録も無ければ隔週配信に落とす。
	pushQuietAfterUnclicked = 4

	// pushQuietRecentScanLimit は隔週判定で遡る配達ログの件数。
	// 1回の配信で端末数ぶんの行ができるため、回数より多めに読む。
	pushQuietRecentScanLimit = 20
)

// isPushUnresponsive は「直近 pushQuietAfterUnclicked 回の campaign の push をすべてタップしておらず、
// その間に記録もしていない」かを返す。配達ログは端末ごとに行ができるため、週(月曜)単位に畳んでから数える。
// lastRecordedWeek はそのユーザーの最終記録週(月曜)。一度も記録していなければゼロ値を渡す。
func isPushUnresponsive(
	ctx context.Context,
	deliveryRepo repository.PushDeliveryInterface,
	userId string,
	campaign string,
	lastRecordedWeek time.Time,
	thisMonday time.Time,
) (bool, error) {
	deliveries, err := deliveryRepo.FindRecentByUserIdAndCampaign(ctx, userId, campaign, pushQuietRecentScanLimit)
	if err != nil {
		return false, err
	}

	clickedByWeek := map[time.Time]bool{}
	var weeks []time.Time
	for _, d := range deliveries {
		week := mondayOf(d.CreatedAt)
		if _, ok := clickedByWeek[week]; !ok {
			clickedByWeek[week] = false
			weeks = append(weeks, week)
		}
		if d.IsClicked() {
			clickedByWeek[week] = true
		}
	}

	if len(weeks) < pushQuietAfterUnclicked {
		return false, nil
	}
	for _, week := range weeks[:pushQuietAfterUnclicked] {
		if clickedByWeek[week] {
			return false, nil
		}
	}

	// 直近の配信期間中に記録があれば「反応が無い」とはみなさない(タップせずに直接開いた可能性)
	quietSince := thisMonday.AddDate(0, 0, -7*pushQuietAfterUnclicked)
	return lastRecordedWeek.Before(quietSince), nil
}

// weekParityEpoch は週の偶奇を数える起点(1970-01-05 は月曜)。
// ISO 週番号だと年またぎで 53→1 と奇数が続き、隔週対象者が2週連続で飛ぶことがあるため、
// 起点からの経過週数で数える。
var weekParityEpoch = time.Date(1970, 1, 5, 0, 0, 0, 0, time.UTC)

// isEvenWeek は monday(週の月曜0時)が起点から偶数週目かを返す。隔週配信の週を決めるのに使う。
// ローカル時刻の月曜0時を UTC の暦日に読み替えて日数を数えるため、タイムゾーンに依存しない。
func isEvenWeek(monday time.Time) bool {
	day := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	weeks := int(day.Sub(weekParityEpoch).Hours()/24) / 7
	return weeks%2 == 0
}
