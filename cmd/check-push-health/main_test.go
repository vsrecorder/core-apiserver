package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func TestBuildSlackMessage(t *testing.T) {
	// 通知を見た人が「どの platform が」「どれだけ」「何を疑えばいいか」まで
	// 分かるようにする。ステータスコードだけ出しても次の一手が決まらない
	t.Run("正常系_全滅したplatformと次に見る場所を載せる", func(t *testing.T) {
		message := buildSlackMessage(7, []*usecase.PushHealthProblem{
			{
				Stat: &entity.PushHealthStat{
					Platform: entity.PushPlatformIOSPWA, Total: 7, Sent: 0, TopFailureStatusCode: 403,
				},
				Severity: usecase.PushHealthSeverityCritical,
			},
		})

		require.Contains(t, message, "直近7日")
		require.Contains(t, message, entity.PushPlatformIOSPWA)
		require.Contains(t, message, "0/7 件成功 (0%)")
		require.Contains(t, message, "1件も届いていない")
		require.Contains(t, message, "403")
		require.Contains(t, message, "verify-vapid-keys")
	})

	t.Run("正常系_成功率が落ちただけなら全滅とは書かない", func(t *testing.T) {
		message := buildSlackMessage(1, []*usecase.PushHealthProblem{
			{
				Stat: &entity.PushHealthStat{
					Platform: entity.PushPlatformAndroid, Total: 10, Sent: 4, TopFailureStatusCode: 410,
				},
				Severity: usecase.PushHealthSeverityWarning,
			},
		})

		require.Contains(t, message, "4/10 件成功 (40%)")
		require.NotContains(t, message, "1件も届いていない")
		require.Contains(t, message, "購読切れ")
	})

	t.Run("正常系_失敗のステータスが無ければヒント行を出さない", func(t *testing.T) {
		message := buildSlackMessage(7, []*usecase.PushHealthProblem{
			{
				Stat: &entity.PushHealthStat{
					Platform: entity.PushPlatformDesktop, Total: 5, Sent: 0, TopFailureStatusCode: 0,
				},
				Severity: usecase.PushHealthSeverityCritical,
			},
		})

		require.NotContains(t, message, "最多の失敗")
	})
}

func TestHintOf(t *testing.T) {
	t.Run("正常系_ステータスごとに疑う場所を返す", func(t *testing.T) {
		require.Contains(t, hintOf(403), "VAPID")
		require.Contains(t, hintOf(410), "購読切れ")
		require.Contains(t, hintOf(404), "購読切れ")
		// 知らないコードでも、次に見る場所は示す
		require.NotEmpty(t, hintOf(500))
	})
}

func TestFormatRate(t *testing.T) {
	require.Equal(t, "0%", formatRate(0))
	require.Equal(t, "40%", formatRate(0.4))
	require.Equal(t, "100%", formatRate(1))
}
