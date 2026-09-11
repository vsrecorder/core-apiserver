package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

/*
 * push が「届いているか」をプラットフォーム別に見張る。
 *
 * push の失敗は静かに起きる。配信バッチは送出に失敗しても通知は残す設計(D2)なので、
 * バッチのログは正常に見えるし、アプリ内通知も普通に増える。そのうえプッシュサービスごとに
 * VAPID の検証の厳しさが違い、Apple は JWT を厳密に見るのに対して FCM のレガシー endpoint は
 * 署名を検証せず 201 を返す。この非対称性のせいで、設定を1つ間違えると
 * 「Android では成功しているのに iOS だけ一通も届かない」という壊れ方をする。
 *
 * 実際、VAPID の subject が二重の mailto: になっていた時期に、iOS への配信が11日間
 * 全滅していたのに誰も気付けなかった。全体の成功率では Android の成功に薄まって見える。
 * だから platform 別に、成功が1件も無い状態を異常として拾う。
 */

const (
	// pushHealthMinDeliveries は判定に必要な最低配達数。
	// 端末1台の一時的な失敗で誤報を出さないための下限。
	pushHealthMinDeliveries = 3

	// pushHealthWarnSuccessRate はこれを下回ると警告する成功率。
	// 全滅ほどではないが、半分以上落ちていれば設定か鍵の疑いがある。
	pushHealthWarnSuccessRate = 0.5
)

// PushHealthSeverity は検知した異常の重さ。
type PushHealthSeverity string

const (
	// PushHealthSeverityCritical は配達しているのに成功が1件も無い状態。
	PushHealthSeverityCritical PushHealthSeverity = "critical"
	// PushHealthSeverityWarning は成功率が大きく落ちている状態。
	PushHealthSeverityWarning PushHealthSeverity = "warning"
)

// PushHealthProblem は異常と判定された platform とその内訳。
type PushHealthProblem struct {
	Stat     *entity.PushHealthStat
	Severity PushHealthSeverity
}

type PushHealthInterface interface {
	// Check は since 以降の配達を platform 別に集計し、集計結果と異常の一覧を返す。
	Check(ctx context.Context, since time.Time) ([]*entity.PushHealthStat, []*PushHealthProblem, error)
}

type PushHealth struct {
	deliveryRepository repository.PushDeliveryInterface
}

func NewPushHealth(
	deliveryRepository repository.PushDeliveryInterface,
) PushHealthInterface {
	return &PushHealth{deliveryRepository}
}

func (u *PushHealth) Check(
	ctx context.Context,
	since time.Time,
) ([]*entity.PushHealthStat, []*PushHealthProblem, error) {
	stats, err := u.deliveryRepository.AggregateHealthByPlatformSince(ctx, since)
	if err != nil {
		logError(ctx, err)
		return nil, nil, err
	}

	problems := make([]*PushHealthProblem, 0)
	for _, stat := range stats {
		// 配達が少ないうちは判定しない。新しい platform や、たまたま1件だけ失敗した
		// 端末で通知が飛ぶと、本当に壊れたときの通知が埋もれる
		if stat.Total < pushHealthMinDeliveries {
			continue
		}

		switch {
		case stat.Sent == 0:
			problems = append(problems, &PushHealthProblem{Stat: stat, Severity: PushHealthSeverityCritical})
		case stat.SuccessRate() < pushHealthWarnSuccessRate:
			problems = append(problems, &PushHealthProblem{Stat: stat, Severity: PushHealthSeverityWarning})
		}
	}

	return stats, problems, nil
}
