package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func setup4TestUserAcquisitionUsecase(t *testing.T) (UserAcquisitionInterface, *mock_repository.MockUserAcquisitionInterface) {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserAcquisitionInterface(mockCtrl)

	return NewUserAcquisition(mockRepository), mockRepository
}

func TestUserAcquisitionUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	// 着地時刻の妥当性判定が現在時刻に依存するため固定する
	now := time.Date(2026, 8, 31, 21, 0, 0, 0, time.Local)

	t.Run("Record", func(t *testing.T) {
		t.Run("正常系_UTMを正規化して保存する", func(t *testing.T) {
			overrideTimeNow(t, now)
			usecase, mockRepository := setup4TestUserAcquisitionUsecase(t)

			var saved *entity.UserAcquisition
			mockRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, e *entity.UserAcquisition) error {
					saved = e
					return nil
				},
			)

			err := usecase.Record(context.Background(), uid, &UserAcquisitionRecordParam{
				Source:      "X",
				Medium:      "Social",
				Campaign:    "HOWTO_CTA",
				Content:     "20260831a",
				Referrer:    "https://t.co/EAbruFy36h",
				LandingPath: "/records/quick?utm_source=x",
				LandingAt:   "2026-08-30T12:00:00Z",
			})

			require.NoError(t, err)
			require.Equal(t, uid, saved.UserId)
			require.Equal(t, "x", saved.Source)
			require.Equal(t, "social", saved.Medium)
			require.Equal(t, entity.AcquisitionCampaignHowtoCta, saved.Campaign)
			require.Equal(t, "20260831a", saved.Content)
			require.Equal(t, "t.co", saved.Referrer)
			require.Equal(t, "/records/quick", saved.LandingPath)
			require.False(t, saved.SourceInferred)
			require.Equal(t, now, saved.CreatedAt)
			require.Equal(t, now, saved.UpdatedAt)
			// 保存先は JST の壁時計を持つ timestamp 列なので Local に寄せる
			require.True(t, saved.LandingAt.Equal(time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)))
			require.Equal(t, time.Local, saved.LandingAt.Location())
		})

		t.Run("正常系_UTMが無ければリファラからチャネルを推定する", func(t *testing.T) {
			overrideTimeNow(t, now)
			usecase, mockRepository := setup4TestUserAcquisitionUsecase(t)

			var saved *entity.UserAcquisition
			mockRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, e *entity.UserAcquisition) error {
					saved = e
					return nil
				},
			)

			err := usecase.Record(context.Background(), uid, &UserAcquisitionRecordParam{
				Referrer: "https://t.co/EAbruFy36h",
			})

			require.NoError(t, err)
			require.Equal(t, "x", saved.Source)
			require.Equal(t, "referral", saved.Medium)
			// 推定値は確定値と混ぜて数えられないよう印を付ける
			require.True(t, saved.SourceInferred)
		})

		t.Run("正常系_utm_sourceがあれば推定で上書きしない", func(t *testing.T) {
			overrideTimeNow(t, now)
			usecase, mockRepository := setup4TestUserAcquisitionUsecase(t)

			var saved *entity.UserAcquisition
			mockRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, e *entity.UserAcquisition) error {
					saved = e
					return nil
				},
			)

			err := usecase.Record(context.Background(), uid, &UserAcquisitionRecordParam{
				Source:   "note",
				Medium:   "social",
				Referrer: "https://t.co/EAbruFy36h",
			})

			require.NoError(t, err)
			require.Equal(t, "note", saved.Source)
			require.Equal(t, "social", saved.Medium)
			require.False(t, saved.SourceInferred)
		})

		t.Run("正常系_何も判明しなければ保存しない", func(t *testing.T) {
			overrideTimeNow(t, now)
			usecase, mockRepository := setup4TestUserAcquisitionUsecase(t)

			// Create が呼ばれないことを EXPECT の不在で確認する
			mockRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			err := usecase.Record(context.Background(), uid, &UserAcquisitionRecordParam{
				LandingPath: "/",
				LandingAt:   "2026-08-30T12:00:00Z",
			})

			require.NoError(t, err)
		})

		t.Run("正常系_現実的でない着地時刻は捨てるが流入元は保存する", func(t *testing.T) {
			overrideTimeNow(t, now)
			usecase, mockRepository := setup4TestUserAcquisitionUsecase(t)

			for _, landingAt := range []string{
				"",
				"not a time",
				"2027-08-31T00:00:00Z", // 未来
				"2020-01-01T00:00:00Z", // Cookie の寿命を大きく超える過去
			} {
				var saved *entity.UserAcquisition
				mockRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, e *entity.UserAcquisition) error {
						saved = e
						return nil
					},
				)

				err := usecase.Record(context.Background(), uid, &UserAcquisitionRecordParam{
					Source:    "x",
					LandingAt: landingAt,
				})

				require.NoError(t, err, landingAt)
				require.Equal(t, "x", saved.Source, landingAt)
				require.True(t, saved.LandingAt.IsZero(), landingAt)
			}
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			overrideTimeNow(t, now)
			usecase, mockRepository := setup4TestUserAcquisitionUsecase(t)

			expected := errors.New("failed to create")
			mockRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(expected)

			err := usecase.Record(context.Background(), uid, &UserAcquisitionRecordParam{Source: "x"})

			require.ErrorIs(t, err, expected)
		})
	})
}
