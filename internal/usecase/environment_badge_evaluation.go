package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type EnvironmentBadgeEvaluationInterface interface {
	// EvaluateOnMatchCreated は対戦結果作成時、対戦日(basisTime)が属する環境の初回対戦バッジを
	// 判定する。新規獲得があればその Environment を返す(お祝いポップアップ表示用)。該当する
	// 環境が無い、または既に獲得済みの場合は nil を返す。対戦作成自体を失敗させないため、
	// 環境が見つからないことはエラー扱いにしない。
	EvaluateOnMatchCreated(
		ctx context.Context,
		userId string,
		match *entity.Match,
		basisTime time.Time,
	) (*entity.Environment, error)

	// NotifyAchieved は環境バッジ獲得の新規通知を作成し、生成した通知IDを返す。バックフィル
	// ツールなど、EvaluateOnMatchCreated を経由せずバッジを直接付与する経路から呼び出すために
	// 公開している。createdAtには達成条件に使ったmatchのCreatedAtを渡す(通知の
	// created_atはuser_environment_badges.created_atと同じ値にするため)。isReadには
	// 通知の既読状態を渡す(バックフィルツールが新規作成する通知は、対戦時点まで遡って
	// 付与するものであり通知ベルに新着として目立たせたくないためtrueを渡す。一方
	// EvaluateOnMatchCreatedからのリアルタイム通知はfalseを渡す)。
	NotifyAchieved(
		ctx context.Context,
		userId string,
		env *entity.Environment,
		createdAt time.Time,
		isRead bool,
	) (string, error)

	// UpdateAchievedNotification は既存の環境バッジ通知の内容を最新の状態(環境名・created_at)で
	// 上書きする。バックフィルツールが判定基準の変更等で再計算した際、新規通知を積み増すのでは
	// なく以前の通知を上書きするために使う。is_read は true にする(overwriteされた通知を
	// 新着として通知ベルに目立たせないため)。該当する通知が無い場合は apperror.ErrRecordNotFound
	// を返す。
	UpdateAchievedNotification(
		ctx context.Context,
		notificationId string,
		env *entity.Environment,
		createdAt time.Time,
	) error
}

type EnvironmentBadgeEvaluation struct {
	environmentRepo          repository.EnvironmentInterface
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface
	notificationRepo         repository.NotificationInterface
	transactionManager       repository.TransactionManager
}

func NewEnvironmentBadgeEvaluation(
	environmentRepo repository.EnvironmentInterface,
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface,
	notificationRepo repository.NotificationInterface,
	transactionManager repository.TransactionManager,
) EnvironmentBadgeEvaluationInterface {
	return &EnvironmentBadgeEvaluation{
		environmentRepo:          environmentRepo,
		userEnvironmentBadgeRepo: userEnvironmentBadgeRepo,
		notificationRepo:         notificationRepo,
		transactionManager:       transactionManager,
	}
}

// environmentBadgeNotificationContent は環境バッジ通知のタイトル・本文を組み立てる。
// 新規作成(NotifyAchieved)・上書き(UpdateAchievedNotification)の両方で同じ内容にするため共通化している。
func environmentBadgeNotificationContent(env *entity.Environment) (title string, body string) {
	return "バッジを獲得しました", fmt.Sprintf("『%s』環境で対戦をしました！", env.Title)
}

func (u *EnvironmentBadgeEvaluation) NotifyAchieved(
	ctx context.Context,
	userId string,
	env *entity.Environment,
	createdAt time.Time,
	isRead bool,
) (string, error) {
	id, err := generateId()
	if err != nil {
		return "", err
	}

	title, body := environmentBadgeNotificationContent(env)

	notification := entity.NewNotification(
		id,
		createdAt,
		userId,
		NotificationCategoryBadge,
		title,
		body,
		notificationLinkUrlForBadge,
	)
	notification.IsRead = isRead

	if err := u.notificationRepo.Save(ctx, notification); err != nil {
		return "", err
	}

	return id, nil
}

func (u *EnvironmentBadgeEvaluation) UpdateAchievedNotification(
	ctx context.Context,
	notificationId string,
	env *entity.Environment,
	createdAt time.Time,
) error {
	title, body := environmentBadgeNotificationContent(env)

	return u.notificationRepo.UpdateContent(ctx, notificationId, createdAt, title, body, true)
}

func (u *EnvironmentBadgeEvaluation) EvaluateOnMatchCreated(
	ctx context.Context,
	userId string,
	match *entity.Match,
	basisTime time.Time,
) (*entity.Environment, error) {
	env, err := u.environmentRepo.FindByDate(ctx, basisTime)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	existing, err := u.userEnvironmentBadgeRepo.FindByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	for _, ub := range existing {
		if ub.EnvironmentId == env.ID {
			return nil, nil
		}
	}

	// created_atは達成条件に使ったmatch自体のCreatedAt(basisTimeではない。basisTimeは
	// achieved_atにのみ使う)。notificationsのcreated_atもこれと同じ値にする。
	userEnvironmentBadge := entity.NewUserEnvironmentBadge(
		userId,
		env.ID,
		match.RecordId,
		"",
		basisTime,
		match.CreatedAt,
	)

	// バッジ保存と通知作成を1つのトランザクションにまとめる。通知作成だけが失敗して
	// バッジのみ保存された状態になると、以後の判定は既存バッジを検知して素通りしてしまい、
	// 通知が永久に作られなくなるため。
	err = u.transactionManager.Do(ctx, func(ctx context.Context) error {
		notificationId, err := u.NotifyAchieved(ctx, userId, env, match.CreatedAt, false)
		if err != nil {
			return err
		}
		userEnvironmentBadge.NotificationId = notificationId

		return u.userEnvironmentBadgeRepo.Save(ctx, userEnvironmentBadge)
	})
	if err != nil {
		return nil, err
	}

	return env, nil
}
