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

// userUsecaseMocks は User が抱えるリポジトリのモック一式。
// 退会で消すテーブルが増えるたびにテスト関数の引数が伸びるのを避けるため、
// まとめて1つの構造体で持ち回る。
type userUsecaseMocks struct {
	user                 *mock_repository.MockUserInterface
	record               *mock_repository.MockRecordInterface
	deck                 *mock_repository.MockDeckInterface
	deckCode             *mock_repository.MockDeckCodeInterface
	tag                  *mock_repository.MockTagInterface
	userFavoriteDeck     *mock_repository.MockUserFavoriteDeckInterface
	userStreak           *mock_repository.MockUserStreakInterface
	userDailyActivity    *mock_repository.MockUserDailyActivityInterface
	userBadge            *mock_repository.MockUserBadgeInterface
	userEnvironmentBadge *mock_repository.MockUserEnvironmentBadgeInterface
	notification         *mock_repository.MockNotificationInterface
	userPlayer           *mock_repository.MockUserPlayerInterface
	pushSubscription     *mock_repository.MockPushSubscriptionInterface
	pushDelivery         *mock_repository.MockPushDeliveryInterface
}

func newUserUsecaseMocks(ctrl *gomock.Controller) *userUsecaseMocks {
	return &userUsecaseMocks{
		user:                 mock_repository.NewMockUserInterface(ctrl),
		record:               mock_repository.NewMockRecordInterface(ctrl),
		deck:                 mock_repository.NewMockDeckInterface(ctrl),
		deckCode:             mock_repository.NewMockDeckCodeInterface(ctrl),
		tag:                  mock_repository.NewMockTagInterface(ctrl),
		userFavoriteDeck:     mock_repository.NewMockUserFavoriteDeckInterface(ctrl),
		userStreak:           mock_repository.NewMockUserStreakInterface(ctrl),
		userDailyActivity:    mock_repository.NewMockUserDailyActivityInterface(ctrl),
		userBadge:            mock_repository.NewMockUserBadgeInterface(ctrl),
		userEnvironmentBadge: mock_repository.NewMockUserEnvironmentBadgeInterface(ctrl),
		notification:         mock_repository.NewMockNotificationInterface(ctrl),
		userPlayer:           mock_repository.NewMockUserPlayerInterface(ctrl),
		pushSubscription:     mock_repository.NewMockPushSubscriptionInterface(ctrl),
		pushDelivery:         mock_repository.NewMockPushDeliveryInterface(ctrl),
	}
}

// expectDeleteAllUserData は「退会で消すテーブルすべてに削除が1回ずつ走る」ことを期待する。
// 退会処理から消し漏らしたリポジトリがあると、その EXPECT が消化されず gomock が落ちる。
// 削除対象を増やしたらここにも足すこと。
func (m *userUsecaseMocks) expectDeleteAllUserData(ctx context.Context, id string) {
	m.record.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.deck.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.deckCode.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.tag.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.userFavoriteDeck.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.userStreak.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.userDailyActivity.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.userBadge.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.userEnvironmentBadge.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.notification.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.userPlayer.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.pushSubscription.EXPECT().DeleteByUserId(ctx, id).Return(nil)
	m.pushDelivery.EXPECT().DeleteByUserId(ctx, id).Return(nil)
}

func TestUserUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	m := newUserUsecaseMocks(mockCtrl)
	mockTransactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	mockTransactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()
	usecase := NewUser(
		m.user,
		m.record,
		m.deck,
		m.deckCode,
		m.tag,
		m.userFavoriteDeck,
		m.userStreak,
		m.userDailyActivity,
		m.userBadge,
		m.userEnvironmentBadge,
		m.notification,
		m.userPlayer,
		m.pushSubscription,
		m.pushDelivery,
		mockTransactionManager,
		stubBadgeEvaluation{},
	)

	for scenario, fn := range map[string]func(
		t *testing.T,
		m *userUsecaseMocks,
		usecase UserInterface,
	){
		"FindById": test_UserUsecase_FindById,
		"Create":   test_UserUsecase_Create,
		"Update":   test_UserUsecase_Update,
		"Delete":   test_UserUsecase_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, m, usecase)
		})
	}
}

func test_UserUsecase_FindById(t *testing.T, m *userUsecaseMocks, usecase UserInterface) {
	mockRepository := m.user

	t.Run("正常系_指定IDのユーザを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		user := &entity.User{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_UserUsecase_Create(t *testing.T, m *userUsecaseMocks, usecase UserInterface) {
	mockRepository := m.user

	t.Run("正常系_未登録IDならユーザを作成する", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)
		mockRepository.EXPECT().IsWithdrawn(context.Background(), id).Return(false, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.NotEmpty(t, ret.ID)
		require.NotEmpty(t, ret.CreatedAt)
		require.Equal(t, name, ret.Name)
		require.Equal(t, imageURL, ret.ImageURL)
	})

	t.Run("異常系_既存IDならErrAlreadyExistsを返す", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		user := &entity.User{
			ID:        id,
			CreatedAt: createdAt,
			Name:      name,
			ImageURL:  imageURL,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)

		ret, err := usecase.Create(context.Background(), param)

		require.ErrorIs(t, err, apperror.ErrAlreadyExists)
		require.Empty(t, ret)
	})

	t.Run("異常系_存在確認でNotFound以外のエラーはそのまま返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Error(t, err)
		require.Empty(t, ret)
	})

	// 退会済みのユーザはFindByIdからは見えないため、そのままSaveすると
	// UPDATEがdeleted_at IS NULLに阻まれて0件更新のまま成功扱いになってしまう。
	// Saveまで到達せずErrWithdrawnで弾かれることを確認する。
	t.Run("異常系_退会済みIDならErrWithdrawnを返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)
		mockRepository.EXPECT().IsWithdrawn(context.Background(), id).Return(true, nil)

		ret, err := usecase.Create(context.Background(), param)

		require.ErrorIs(t, err, apperror.ErrWithdrawn)
		require.Empty(t, ret)
	})

	t.Run("異常系_退会済み確認でエラーならそのまま返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)
		mockRepository.EXPECT().IsWithdrawn(context.Background(), id).Return(false, errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_UserUsecase_Update(t *testing.T, m *userUsecaseMocks, usecase UserInterface) {
	mockRepository := m.user

	t.Run("正常系_取得したユーザにパラメータを反映して保存する", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserUpdateParam(
			name,
			imageURL,
		)

		user := &entity.User{
			ID:        id,
			CreatedAt: createdAt,
			Name:      name,
			ImageURL:  imageURL,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.NoError(t, err)
		require.NotEmpty(t, ret.ID)
		require.NotEmpty(t, ret.CreatedAt)
		require.Equal(t, name, ret.Name)
		require.Equal(t, imageURL, ret.ImageURL)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserUpdateParam(
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Update(context.Background(), id, param)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_NotFound以外の取得エラーもそのまま返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserUpdateParam(
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.Update(context.Background(), id, param)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_UserUsecase_Delete(t *testing.T, m *userUsecaseMocks, usecase UserInterface) {
	mockDeckCodeRepository := m.deckCode
	mockDeckRepository := m.deck
	mockRecordRepository := m.record
	mockRepository := m.user

	t.Run("正常系_ユーザに紐づくデータをすべて削除する", func(t *testing.T) {
		id, _ := generateId()

		// 退会で消すテーブルは、件数によらず種類ごとに1回ずつ呼ぶ。
		// 消し漏らしたリポジトリがあると、その EXPECT が消化されず gomock が落ちる。
		m.expectDeleteAllUserData(context.Background(), id)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		err := usecase.Delete(context.Background(), id)

		require.NoError(t, err)
	})

	t.Run("異常系_対戦記録の削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})

	t.Run("異常系_デッキの削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})

	t.Run("異常系_デッキコードの削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckCodeRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})

	t.Run("異常系_タグの削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckCodeRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.tag.EXPECT().DeleteByUserId(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})

	t.Run("異常系_通知の削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckCodeRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.tag.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userFavoriteDeck.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userStreak.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userDailyActivity.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userBadge.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userEnvironmentBadge.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.notification.EXPECT().DeleteByUserId(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})

	t.Run("異常系_プレイヤーIDの紐付けの削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		mockDeckCodeRepository.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.tag.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userFavoriteDeck.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userStreak.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userDailyActivity.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userBadge.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userEnvironmentBadge.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.notification.EXPECT().DeleteByUserId(context.Background(), id).Return(nil)
		m.userPlayer.EXPECT().DeleteByUserId(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})

	t.Run("異常系_ユーザ本体の削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		m.expectDeleteAllUserData(context.Background(), id)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})
}
