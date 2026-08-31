package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

type userGymUsecaseMocks struct {
	userGym       *mock_repository.MockUserGymInterface
	shop          *mock_repository.MockShopInterface
	officialEvent *mock_repository.MockOfficialEventInterface
}

func setup4TestUserGymUsecase(t *testing.T) (UserGymInterface, *userGymUsecaseMocks) {
	mockCtrl := gomock.NewController(t)

	m := &userGymUsecaseMocks{
		userGym:       mock_repository.NewMockUserGymInterface(mockCtrl),
		shop:          mock_repository.NewMockShopInterface(mockCtrl),
		officialEvent: mock_repository.NewMockOfficialEventInterface(mockCtrl),
	}

	// 登録は1トランザクションで行う。ここでは中の関数をそのまま実行する。
	transactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	transactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()

	return NewUserGym(m.userGym, m.shop, m.officialEvent, transactionManager), m
}

func newTestShop(id uint) *entity.Shop {
	return entity.NewShop(id, "カードショップ", "194-0013", 13, "東京都", "町田市原町田", "", "", "")
}

func newTestUserGymViews(shopIds ...uint) []*entity.UserGymView {
	views := make([]*entity.UserGymView, 0, len(shopIds))
	for i, shopId := range shopIds {
		views = append(views, entity.NewUserGymView(
			newTestShop(shopId),
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i)*time.Minute),
		))
	}

	return views
}

func TestUserGymUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_登録順のMyジムをそのまま返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			views := newTestUserGymViews(10317, 10318)
			m.userGym.EXPECT().FindByUserId(context.Background(), uid).Return(views, nil)

			ret, err := usecase.Find(context.Background(), uid)

			require.NoError(t, err)
			require.Len(t, ret, 2)
			require.Equal(t, uint(10317), ret[0].Shop.ID)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_空きがあれば登録して店舗情報つきで返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			now := time.Date(2026, 8, 31, 19, 0, 0, 0, time.UTC)
			overrideTimeNow(t, now)

			m.shop.EXPECT().FindById(context.Background(), uint(10317)).Return(newTestShop(10317), nil)
			m.userGym.EXPECT().LockByUserId(context.Background(), uid).Return(nil)
			m.userGym.EXPECT().FindByUserId(context.Background(), uid).Return([]*entity.UserGymView{}, nil)
			m.userGym.EXPECT().Create(context.Background(), entity.NewUserGym(uid, 10317, now)).Return(nil)

			ret, err := usecase.Create(context.Background(), uid, 10317)

			require.NoError(t, err)
			require.Equal(t, uint(10317), ret.Shop.ID)
			require.Equal(t, now, ret.CreatedAt)
		})

		t.Run("異常系_上限に達していれば登録せずErrTooManyUserGymsを返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			// MaxUserGymsPerUser 件ちょうど登録されている状態を作る
			shopIds := make([]uint, 0, MaxUserGymsPerUser)
			for i := 0; i < MaxUserGymsPerUser; i++ {
				shopIds = append(shopIds, uint(20000+i))
			}

			m.shop.EXPECT().FindById(context.Background(), uint(10317)).Return(newTestShop(10317), nil)
			m.userGym.EXPECT().LockByUserId(context.Background(), uid).Return(nil)
			m.userGym.EXPECT().FindByUserId(context.Background(), uid).Return(newTestUserGymViews(shopIds...), nil)
			// 古いものを押し出さない(Create も Delete も呼ばれない)

			ret, err := usecase.Create(context.Background(), uid, 10317)

			require.Nil(t, ret)
			require.ErrorIs(t, err, apperror.ErrTooManyUserGyms)
		})

		t.Run("異常系_登録済みの店舗ならErrAlreadyExistsを返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			m.shop.EXPECT().FindById(context.Background(), uint(10317)).Return(newTestShop(10317), nil)
			m.userGym.EXPECT().LockByUserId(context.Background(), uid).Return(nil)
			m.userGym.EXPECT().FindByUserId(context.Background(), uid).Return(newTestUserGymViews(10317), nil)

			ret, err := usecase.Create(context.Background(), uid, 10317)

			require.Nil(t, ret)
			require.ErrorIs(t, err, apperror.ErrAlreadyExists)
		})

		t.Run("異常系_存在しない店舗なら登録前にErrRecordNotFoundを返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			// 一覧を引く前に弾く(外部キー違反で500にしない)
			m.shop.EXPECT().FindById(context.Background(), uint(99999999)).Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.Create(context.Background(), uid, 99999999)

			require.Nil(t, ret)
			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_指定した店舗の登録を解除する", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			m.userGym.EXPECT().Delete(context.Background(), uid, uint(10317)).Return(nil)

			require.NoError(t, usecase.Delete(context.Background(), uid, 10317))
		})
	})

	t.Run("FindOfficialEvents", func(t *testing.T) {
		startDate := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)

		t.Run("正常系_登録店舗のイベントをMyジムと併せて返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			views := newTestUserGymViews(10317, 10318)
			officialEvents := []*entity.OfficialEvent{{ID: 606466, ShopId: 10317}}

			m.userGym.EXPECT().FindByUserId(context.Background(), uid).Return(views, nil)
			m.officialEvent.EXPECT().FindByShopIds(
				context.Background(), []uint{10317, 10318}, startDate, endDate,
			).Return(officialEvents, nil)

			retViews, retEvents, err := usecase.FindOfficialEvents(context.Background(), uid, startDate, endDate)

			require.NoError(t, err)
			require.Len(t, retViews, 2)
			require.Len(t, retEvents, 1)
			require.Equal(t, uint(606466), retEvents[0].ID)
		})

		t.Run("正常系_Myジムが無ければイベントを引かずに空で返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			m.userGym.EXPECT().FindByUserId(context.Background(), uid).Return([]*entity.UserGymView{}, nil)
			// officialEvent.FindByShopIds は呼ばれない

			retViews, retEvents, err := usecase.FindOfficialEvents(context.Background(), uid, startDate, endDate)

			require.NoError(t, err)
			require.Empty(t, retViews)
			require.Empty(t, retEvents)
		})

		t.Run("異常系_イベントの取得に失敗したらエラーを返す", func(t *testing.T) {
			usecase, m := setup4TestUserGymUsecase(t)

			m.userGym.EXPECT().FindByUserId(context.Background(), uid).Return(newTestUserGymViews(10317), nil)
			m.officialEvent.EXPECT().FindByShopIds(
				context.Background(), []uint{10317}, startDate, endDate,
			).Return(nil, errors.New("error"))

			retViews, retEvents, err := usecase.FindOfficialEvents(context.Background(), uid, startDate, endDate)

			require.Error(t, err)
			require.Nil(t, retViews)
			require.Nil(t, retEvents)
		})
	})
}
