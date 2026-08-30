package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubPushNotifier は PushNotifierInterface の手書きスタブ。
// usecase パッケージのテストは mock_usecase を import できない(import cycle)ため、
// 他の usecase を差し替えるときは stubBadgeEvaluation と同じく手書きにする。
type stubPushDeliverCall struct {
	notification *entity.Notification
	campaign     string
}

type stubPushNotifier struct {
	calls []stubPushDeliverCall
	sent  int
	err   error
}

func (s *stubPushNotifier) Deliver(_ context.Context, notification *entity.Notification, campaign string) (int, error) {
	s.calls = append(s.calls, stubPushDeliverCall{notification: notification, campaign: campaign})
	return s.sent, s.err
}

// campaigns は Deliver が呼ばれた順のキャンペーン名を返す(呼ばれていなければ nil)。
func (s *stubPushNotifier) campaigns() []string {
	var names []string
	for _, c := range s.calls {
		names = append(names, c.campaign)
	}
	return names
}
