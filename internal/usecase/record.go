package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type RecordParam struct {
	officialEventId   uint
	tonamelEventId    string
	friendId          string
	unofficialEventId string
	userId            string
	deckId            string
	deckCodeId        string
	eventDate         time.Time
	privateFlg        bool
	ignoreStatsFlg    bool
	regulationId      uint
	tcgMeisterURL     string
	memo              string
	// TagIds は付与するタグID。NewRecordParam の引数には含めず、controller が
	// param.TagIds に直接設定する(既存の NewRecordParam 呼び出しを壊さないため。
	// MatchParam.TagIds と同じ扱い)。
	TagIds []string
}

func NewRecordParam(
	officialEventId uint,
	tonamelEventId string,
	friendId string,
	unofficialEventId string,
	userId string,
	deckId string,
	deckCodeId string,
	eventDate time.Time,
	privateFlg bool,
	ignoreStatsFlg bool,
	regulationId uint,
	tcgMeisterURL string,
	memo string,
) *RecordParam {
	return &RecordParam{
		officialEventId:   officialEventId,
		tonamelEventId:    tonamelEventId,
		friendId:          friendId,
		unofficialEventId: unofficialEventId,
		userId:            userId,
		deckId:            deckId,
		deckCodeId:        deckCodeId,
		eventDate:         eventDate,
		privateFlg:        privateFlg,
		ignoreStatsFlg:    ignoreStatsFlg,
		regulationId:      regulationId,
		tcgMeisterURL:     tcgMeisterURL,
		memo:              memo,
	}
}

type RecordInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Record, error)

	Find(
		ctx context.Context,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByTonamelEventId(
		ctx context.Context,
		tonamelEventId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByDeckId(
		ctx context.Context,
		deckId string,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckIdOnCursor(
		ctx context.Context,
		deckId string,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckCodeId(
		ctx context.Context,
		deckCodeId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	Create(
		ctx context.Context,
		param *RecordParam,
	) (*entity.Record, error)

	Update(
		ctx context.Context,
		id string,
		param *RecordParam,
	) (*entity.Record, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type Record struct {
	logger                *slog.Logger
	repository            repository.RecordInterface
	tag                   repository.TagInterface
	badgeEvaluation       BadgeEvaluationInterface
	designationEvaluation DesignationEvaluationInterface
	// tonamelEventRepo は tonamel.com から大会情報を取得する(HTTP)。
	// tonamelEventStore は取得結果をDBへ保存・参照する。
	// 記録作成時に一度だけ取得して保存し、カレンダー等の参照を外部通信なしにする。
	tonamelEventRepo  repository.TonamelEventInterface
	tonamelEventStore repository.TonamelEventStoreInterface
}

func NewRecord(
	logger *slog.Logger,
	repository repository.RecordInterface,
	tag repository.TagInterface,
	badgeEvaluation BadgeEvaluationInterface,
	designationEvaluation DesignationEvaluationInterface,
	tonamelEventRepo repository.TonamelEventInterface,
	tonamelEventStore repository.TonamelEventStoreInterface,
) RecordInterface {
	return &Record{
		logger:                logger,
		repository:            repository,
		tag:                   tag,
		badgeEvaluation:       badgeEvaluation,
		designationEvaluation: designationEvaluation,
		tonamelEventRepo:      tonamelEventRepo,
		tonamelEventStore:     tonamelEventStore,
	}
}

// syncRecordTags は記録について、userId が付与できる有効なタグ(自分のタグ or
// プリセット)だけを残して record_tags を更新し、付与後のタグを返す。
// 挙動は Match.syncMatchTags と同じ。
func (u *Record) syncRecordTags(
	ctx context.Context,
	recordId string,
	userId string,
	tagIds []string,
) ([]*entity.Tag, error) {
	tags, err := u.tag.FindAttachableByIds(ctx, tagIds, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// FindAttachableByIds の戻り順は不定なので、付与順(tagIds)に整列してから採番する。
	orderedTags, attachableTagIds := orderAttachableTagsByIds(tags, tagIds)

	if err := u.tag.ReplaceRecordTags(ctx, recordId, attachableTagIds); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return orderedTags, nil
}

func (u *Record) FindById(
	ctx context.Context,
	id string,
) (*entity.Record, error) {
	record, err := u.repository.FindById(ctx, id)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return record, nil
}

func (u *Record) Find(
	ctx context.Context,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.Find(ctx, limit, offset, eventType)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindOnCursor(
	ctx context.Context,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindOnCursor(ctx, limit, cursorEventDate, cursorCreatedAt, eventType)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByUserId(
	ctx context.Context,
	uid string,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByUserId(ctx, uid, limit, offset, eventType)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByUserIdOnCursor(ctx, uid, limit, cursorEventDate, cursorCreatedAt, eventType)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByOfficialEventId(ctx, officialEventId, limit, offset)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByTonamelEventId(
	ctx context.Context,
	tonamelEventId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByTonamelEventId(ctx, tonamelEventId, limit, offset)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByDeckId(
	ctx context.Context,
	deckId string,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByDeckId(ctx, deckId, limit, offset, eventType)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByDeckIdOnCursor(
	ctx context.Context,
	deckId string,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByDeckIdOnCursor(ctx, deckId, limit, cursorEventDate, cursorCreatedAt, eventType)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByDeckCodeId(
	ctx context.Context,
	deckCodeId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByDeckCodeId(ctx, deckCodeId, limit, offset)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return records, nil
}

// normalizeAndValidateRecordParam は記録の整合性を domain 層の共通関数で検証する。
// controller の middleware と同じルールを使い、middleware を通らない経路でも
// 不整合な記録が保存されないようにする。
// レギュレーション未指定(0)だけは弾かずに既定値へ寄せる(regulation_id を送らない
// 旧クライアントからの記録作成を、DB側の DEFAULT と同じ扱いで受け付けるため)。
func normalizeAndValidateRecordParam(param *RecordParam) error {
	if !entity.IsValidRecordEventSource(entity.RecordEventSource{
		OfficialEventId:   param.officialEventId,
		TonamelEventId:    param.tonamelEventId,
		FriendId:          param.friendId,
		UnofficialEventId: param.unofficialEventId,
	}) {
		return apperror.ErrInvalidRecord
	}

	param.regulationId = entity.NormalizeRegulationId(param.regulationId)
	if !entity.IsValidRegulationId(param.regulationId) {
		return apperror.ErrInvalidRecord
	}

	return nil
}

func (u *Record) Create(
	ctx context.Context,
	param *RecordParam,
) (*entity.Record, error) {
	if err := normalizeAndValidateRecordParam(param); err != nil {
		logError(ctx, err)
		return nil, err
	}

	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	createdAt := time.Now().Local()

	// 称号のtier変化を記録の前後で比較するため、保存前の時点で取得しておく。
	// シーズン範囲が定まらない等でエラーになった場合は、この記録作成では
	// 称号/ランクの通知判定自体を行わない(記録作成そのものは失敗させない)。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, param.userId)

	record := entity.NewRecord(
		id,
		createdAt,
		param.officialEventId,
		param.tonamelEventId,
		param.friendId,
		param.unofficialEventId,
		param.userId,
		param.deckId,
		param.deckCodeId,
		param.eventDate,
		param.privateFlg,
		param.ignoreStatsFlg,
		param.regulationId,
		param.tcgMeisterURL,
		param.memo,
	)
	if param.deckId != "" || param.deckCodeId != "" {
		record.DeckRegisteredAt = &createdAt
	}

	if err := u.repository.Save(ctx, record); err != nil {
		logError(ctx, err)
		return nil, err
	}

	// タグの付与は記録本体とは別テーブルのため Save とは分けて反映する。
	tags, err := u.syncRecordTags(ctx, record.ID, param.userId, param.TagIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	record.Tags = tags

	// Tonamel記録なら、大会情報をこの時点で一度だけ取得してDBへ保存しておく。
	// カレンダー等はこれを参照するだけで済み、表示のたびに外部サイトを引かずに済む。
	u.persistTonamelEvent(ctx, param.tonamelEventId)

	// 通知一覧はcreated_at DESC(新しい順、同値時はid DESC)で表示されるため、後から
	// 生成した通知ほど上に表示される。作成順序を「ユーザバッジ→称号/ランクアップ」に
	// することで、表示順序は下から「ユーザバッジ→称号/ランクアップ」(=上から称号/
	// ランクアップ→ユーザバッジ)になる。
	if _, err := u.badgeEvaluation.EvaluateOnRecordCreated(ctx, param.userId, record); err != nil {
		logError(ctx, err)
		return nil, err
	}

	if tierErr == nil {
		// 通知のcreated_atは対戦日(event_date)ではなく実際の処理時刻を使う。
		// event_dateはユーザ登録直後に過去日を入力されると、登録バッジ通知より
		// 過去のcreated_atになり通知の並び順が崩れるため使わない。
		u.designationEvaluation.NotifyIfTierChanged(ctx, param.userId, beforeTier, record.CreatedAt)
	}

	return record, nil
}

// persistTonamelEvent は Tonamel の大会情報を tonamel_events へ保存する。
//
// すべてベストエフォートで、失敗しても記録作成自体は成功させる(大会情報が
// 無くてもタイトル不明として扱えるため。カレンダー側と同じ寛容な方針)。
//   - tonamelEventId が空なら何もしない。
//   - 既に保存済みなら再取得しない(大会情報は不変で全ユーザー共通のため、
//     別のユーザーや過去の記録作成で保存済みなら外部通信を省ける)。
//   - 未保存なら tonamel.com から取得して保存する。
func (u *Record) persistTonamelEvent(
	ctx context.Context,
	tonamelEventId string,
) {
	if tonamelEventId == "" {
		return
	}

	existing, err := u.tonamelEventStore.FindByIds(ctx, []string{tonamelEventId})
	if err != nil {
		u.logger.WarnContext(
			ctx,
			"failed to look up tonamel event before persisting",
			slog.String("tonamel_event_id", tonamelEventId),
			slog.String("error_message", err.Error()),
		)
		return
	}
	if len(existing) > 0 {
		return
	}

	tonamelEvent, err := u.tonamelEventRepo.FindById(ctx, tonamelEventId)
	if err != nil {
		u.logger.WarnContext(
			ctx,
			"failed to fetch tonamel event for persisting",
			slog.String("tonamel_event_id", tonamelEventId),
			slog.String("error_message", err.Error()),
		)
		return
	}

	if err := u.tonamelEventStore.Save(ctx, tonamelEvent); err != nil {
		u.logger.WarnContext(
			ctx,
			"failed to save tonamel event",
			slog.String("tonamel_event_id", tonamelEventId),
			slog.String("error_message", err.Error()),
		)
	}
}

func (u *Record) Update(
	ctx context.Context,
	id string,
	param *RecordParam,
) (*entity.Record, error) {
	if err := normalizeAndValidateRecordParam(param); err != nil {
		logError(ctx, err)
		return nil, err
	}

	// 指定されたidのRecordが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// 称号のtier変化を更新の前後で比較するため、保存前の時点で取得しておく。
	// デッキ未登録のまま作成した記録に、後からデッキを登録するケースでは、この
	// Updateで初めて称号のrecordカウント対象になりtierが変化しうる(Createと同様)。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, param.userId)

	record := entity.NewRecord(
		id,
		ret.CreatedAt,
		param.officialEventId,
		param.tonamelEventId,
		param.friendId,
		param.unofficialEventId,
		param.userId,
		param.deckId,
		param.deckCodeId,
		param.eventDate,
		param.privateFlg,
		param.ignoreStatsFlg,
		param.regulationId,
		param.tcgMeisterURL,
		param.memo,
	)
	// デッキ未登録のまま作成した記録に、後からデッキを登録した瞬間だけ
	// DeckRegisteredAtを更新する。それ以外(既に登録済み/デッキ以外の編集/
	// デッキ未登録のまま)は更新前の値をそのまま引き継ぐ。
	record.DeckRegisteredAt = ret.DeckRegisteredAt
	if record.DeckRegisteredAt == nil && (param.deckId != "" || param.deckCodeId != "") {
		now := time.Now().Local()
		record.DeckRegisteredAt = &now
	}

	if err := u.repository.Save(ctx, record); err != nil {
		logError(ctx, err)
		return nil, err
	}

	// タグの付与を param.TagIds の集合に合わせて更新する。
	tags, err := u.syncRecordTags(ctx, record.ID, param.userId, param.TagIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	record.Tags = tags

	// 編集で Tonamel記録に変わった/別の大会に付け替えられたケースに追随する。
	// 既に保存済みの大会なら再取得しない(persistTonamelEvent 内で判定)。
	u.persistTonamelEvent(ctx, param.tonamelEventId)

	// 対戦日を動かすと週次ストリークの連続週数が変わりうる(週が埋まって伸びることも、
	// 空いて途切れることもある)。Createで使う updateStreak は加算のみの差分更新で
	// 減少を追えないため、削除時と同じく現存する記録からゼロで作り直す。
	// 対戦日が変わらない更新(メモやデッキの編集)では集計結果も変わらないため呼ばない。
	// 削除時と同じ理由で、再計算の失敗は更新自体の失敗にしない(更新はもう保存済み)。
	if !isSameEventDate(ret.EventDate, record.EventDate) {
		if err := u.badgeEvaluation.EvaluateOnRecordUpdated(ctx, param.userId); err != nil {
			logError(ctx, err)
		}
	}

	if tierErr == nil {
		u.designationEvaluation.NotifyIfTierChanged(ctx, param.userId, beforeTier, time.Now().Local())
		u.designationEvaluation.NotifyIfTierLost(ctx, param.userId, beforeTier)
	}

	return record, nil
}

func (u *Record) Delete(
	ctx context.Context,
	id string,
) error {
	record, err := u.repository.FindById(ctx, id)
	if err != nil {
		logError(ctx, err)
		return err
	}

	// 称号のtier変化を削除の前後で比較するため、削除前の時点で取得しておく。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, record.UserId)

	if err := u.repository.Delete(ctx, id); err != nil {
		logError(ctx, err)
		return err
	}

	// ストリークの再計算に失敗しても、記録の削除自体は完了しているので成功として返す。
	// ここでエラーにすると、消えているのに「削除に失敗」と見えてしまう。ズレは次の
	// 記録の作成・削除・更新での再計算か、repair-streaks で解消できる。
	if err := u.badgeEvaluation.EvaluateOnRecordDeleted(ctx, record.UserId); err != nil {
		logError(ctx, err)
	}

	if tierErr == nil {
		u.designationEvaluation.NotifyIfTierLost(ctx, record.UserId, beforeTier)
	}

	return nil
}
