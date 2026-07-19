package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type GameParam struct {
	GoFirst             bool
	WinningFlg          bool
	YourPrizeCards      uint
	OpponentsPrizeCards uint
	Memo                string
}

type MatchParam struct {
	RecordId             string
	DeckId               string
	DeckCodeId           string
	UserId               string
	OpponentsUserId      string
	BO3Flg               bool
	GroupMatchFlg        bool
	QualifyingRoundFlg   bool
	FinalTournamentFlg   bool
	DefaultVictoryFlg    bool
	DefaultDefeatFlg     bool
	VictoryFlg           bool
	GroupMatchVictoryFlg bool
	OpponentsDeckInfo    string
	Memo                 string
	Games                []*GameParam
	PokemonSprites       []*PokemonSpriteParam
}

func NewGameParam(
	goFirst bool,
	winningFlg bool,
	yourPrizeCards uint,
	opponentsPrizeCards uint,
	memo string,
) *GameParam {
	return &GameParam{
		GoFirst:             goFirst,
		WinningFlg:          winningFlg,
		YourPrizeCards:      yourPrizeCards,
		OpponentsPrizeCards: opponentsPrizeCards,
		Memo:                memo,
	}
}

func NewMatchParam(
	recordId string,
	deckId string,
	deckCodeId string,
	userId string,
	opponentsUserId string,
	bo3Flg bool,
	groupMatchFlg bool,
	qualifyingRoundFlg bool,
	finalTournamentFlg bool,
	defaultVictoryFlg bool,
	defaultDefeatFlg bool,
	victoryFlg bool,
	groupMatchVictoryFlg bool,
	opponentsDeckInfo string,
	memo string,
	games []*GameParam,
	pokemonSprites []*PokemonSpriteParam,
) *MatchParam {
	return &MatchParam{
		RecordId:             recordId,
		DeckId:               deckId,
		DeckCodeId:           deckCodeId,
		UserId:               userId,
		OpponentsUserId:      opponentsUserId,
		BO3Flg:               bo3Flg,
		GroupMatchFlg:        groupMatchFlg,
		QualifyingRoundFlg:   qualifyingRoundFlg,
		FinalTournamentFlg:   finalTournamentFlg,
		DefaultVictoryFlg:    defaultVictoryFlg,
		DefaultDefeatFlg:     defaultDefeatFlg,
		VictoryFlg:           victoryFlg,
		GroupMatchVictoryFlg: groupMatchVictoryFlg,
		OpponentsDeckInfo:    opponentsDeckInfo,
		Memo:                 memo,
		Games:                games,
		PokemonSprites:       pokemonSprites,
	}
}

type MatchInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Match, error)

	FindByRecordId(
		ctx context.Context,
		recordId string,
	) ([]*entity.Match, error)

	FindByUserId(
		ctx context.Context,
		userId string,
		limit int,
	) ([]*entity.Match, error)

	FindLatest(
		ctx context.Context,
		limit int,
	) ([]*entity.Match, error)

	Create(
		ctx context.Context,
		param *MatchParam,
	) (*entity.Match, error)

	Update(
		ctx context.Context,
		id string,
		param *MatchParam,
	) (*entity.Match, error)

	Delete(
		ctx context.Context,
		id string,
	) error

	Reorder(
		ctx context.Context,
		recordId string,
		orders []*entity.MatchOrder,
	) error
}

type Match struct {
	repository            repository.MatchInterface
	recordRepository      repository.RecordInterface
	badgeEvaluation       BadgeEvaluationInterface
	designationEvaluation DesignationEvaluationInterface
	environmentBadgeEval  EnvironmentBadgeEvaluationInterface
}

func NewMatch(
	repository repository.MatchInterface,
	recordRepository repository.RecordInterface,
	badgeEvaluation BadgeEvaluationInterface,
	designationEvaluation DesignationEvaluationInterface,
	environmentBadgeEval EnvironmentBadgeEvaluationInterface,
) MatchInterface {
	return &Match{repository, recordRepository, badgeEvaluation, designationEvaluation, environmentBadgeEval}
}

func (u *Match) FindById(
	ctx context.Context,
	id string,
) (*entity.Match, error) {
	match, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return match, nil
}

func (u *Match) FindByRecordId(
	ctx context.Context,
	recordId string,
) ([]*entity.Match, error) {
	matches, err := u.repository.FindByRecordId(ctx, recordId)

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (u *Match) FindByUserId(
	ctx context.Context,
	userId string,
	limit int,
) ([]*entity.Match, error) {
	matches, err := u.repository.FindByUserId(ctx, userId, limit)

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (u *Match) FindLatest(
	ctx context.Context,
	limit int,
) ([]*entity.Match, error) {
	matches, err := u.repository.FindLatest(ctx, limit)

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (u *Match) Create(
	ctx context.Context,
	param *MatchParam,
) (*entity.Match, error) {
	matchId, err := generateId()
	if err != nil {
		return nil, err
	}

	// 称号のtier変化を対戦結果作成の前後で比較するため、保存前の時点で取得しておく。
	// 称号の「記録数」条件は対戦結果が1件以上紐づく記録のみをカウントするため、
	// 記録作成時点ではなく対戦結果作成時点でtierが上がることが多い。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, param.UserId)

	createdAt := time.Now().Local()

	var games []*entity.Game
	for _, game := range param.Games {
		gameId, err := generateId()
		if err != nil {
			return nil, err
		}

		createdAt := time.Now().Local()

		games = append(
			games,
			entity.NewGame(
				gameId,
				createdAt,
				matchId,
				param.UserId,
				game.GoFirst,
				game.WinningFlg,
				game.YourPrizeCards,
				game.OpponentsPrizeCards,
				game.Memo,
			),
		)
	}

	var pokemonSprites []*entity.PokemonSprite
	for _, pokemonSprite := range param.PokemonSprites {
		pokemonSprites = append(
			pokemonSprites,
			entity.NewPokemonSpriteWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
	}

	match := entity.NewMatch(
		matchId,
		createdAt,
		param.RecordId,
		param.DeckId,
		param.DeckCodeId,
		param.UserId,
		param.OpponentsUserId,
		param.BO3Flg,
		param.GroupMatchFlg,
		param.QualifyingRoundFlg,
		param.FinalTournamentFlg,
		param.DefaultVictoryFlg,
		param.DefaultDefeatFlg,
		param.VictoryFlg,
		param.GroupMatchVictoryFlg,
		param.OpponentsDeckInfo,
		param.Memo,
		games,
		pokemonSprites,
	)

	if err := u.repository.Create(ctx, match); err != nil {
		return nil, err
	}

	// 通知一覧はcreated_at DESC(新しい順、同値時はid DESC)で表示されるため、後から
	// 生成した通知ほど上に表示される。作成順序を「ユーザバッジ→環境バッジ→称号/
	// ランクアップ」にすることで、表示順序は下から「ユーザバッジ→環境バッジ→称号/
	// ランクアップ」(=上から称号/ランクアップ→環境バッジ→ユーザバッジ)になる。
	if _, err := u.badgeEvaluation.EvaluateOnMatchCreated(ctx, param.UserId, match); err != nil {
		return nil, err
	}

	// 環境バッジは公式イベント(OfficialEventId != 0)に紐づく記録のみを対象とする。
	// 環境判定も「対戦結果を入力した日時」ではなく「実際に対戦した日」(紐づくrecordの
	// event_date)を使いたいため、親recordを取得する。取得できない場合(通常発生しない)や
	// 公式イベントでない記録の場合は環境バッジの判定自体を行わない。
	if record, err := u.recordRepository.FindById(ctx, param.RecordId); err == nil && record.OfficialEventId != 0 {
		basisTime := RecordBasisTime(record.EventDate, record.CreatedAt)

		if _, err := u.environmentBadgeEval.EvaluateOnMatchCreated(ctx, param.UserId, match, basisTime); err != nil {
			return nil, err
		}
	}

	if tierErr == nil {
		// 通知のcreated_atは対戦日(event_date)ではなく実際の処理時刻を使う。
		// event_dateを使うと登録直後に過去日の対戦を記録した際、他の通知より
		// 過去のcreated_atになり通知の並び順が崩れるため使わない(basisTimeは環境
		// 判定にのみ使う)。
		u.designationEvaluation.NotifyIfTierChanged(ctx, param.UserId, beforeTier, match.CreatedAt)
	}

	return match, nil
}

func (u *Match) Update(
	ctx context.Context,
	id string,
	param *MatchParam,
) (*entity.Match, error) {
	// 指定されたidのMatchを取得
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	var games []*entity.Game

	// len(ret.Games) <= len(param.Games) の場合は新しくGameが追加されている可能性がある
	if len(ret.Games) <= len(param.Games) {
		for i, game := range param.Games {
			if i < len(ret.Games) { // 既存のGameを上書き
				games = append(
					games,
					entity.NewGame(
						ret.Games[i].ID,
						ret.Games[i].CreatedAt,
						id,
						param.UserId,
						game.GoFirst,
						game.WinningFlg,
						game.YourPrizeCards,
						game.OpponentsPrizeCards,
						game.Memo,
					),
				)
			} else { // 新しくGameを追加
				gameId, err := generateId()
				if err != nil {
					return nil, err
				}

				createdAt := time.Now().Local()

				games = append(
					games,
					entity.NewGame(
						gameId,
						createdAt,
						id,
						param.UserId,
						game.GoFirst,
						game.WinningFlg,
						game.YourPrizeCards,
						game.OpponentsPrizeCards,
						game.Memo,
					),
				)
			}
		}
	} else { // それ以外(len(ret.Games) > len(param.Games))はGameが削除されている
		for i, game := range param.Games {
			games = append(
				games,
				entity.NewGame(
					ret.Games[i].ID,
					ret.Games[i].CreatedAt,
					id,
					param.UserId,
					game.GoFirst,
					game.WinningFlg,
					game.YourPrizeCards,
					game.OpponentsPrizeCards,
					game.Memo,
				),
			)
		}
	}

	var pokemonSprites []*entity.PokemonSprite
	for _, pokemonSprite := range param.PokemonSprites {
		pokemonSprites = append(
			pokemonSprites,
			entity.NewPokemonSpriteWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
	}

	match := entity.NewMatch(
		ret.ID,
		ret.CreatedAt,
		param.RecordId,
		param.DeckId,
		param.DeckCodeId,
		param.UserId,
		param.OpponentsUserId,
		param.BO3Flg,
		param.GroupMatchFlg,
		param.QualifyingRoundFlg,
		param.FinalTournamentFlg,
		param.DefaultVictoryFlg,
		param.DefaultDefeatFlg,
		param.VictoryFlg,
		param.GroupMatchVictoryFlg,
		param.OpponentsDeckInfo,
		param.Memo,
		games,
		pokemonSprites,
	)
	// position は通常の更新では変更しないため、既存値を引き継ぐ
	match.Position = ret.Position

	if err := u.repository.Update(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

func (u *Match) Delete(
	ctx context.Context,
	id string,
) error {
	match, err := u.repository.FindById(ctx, id)
	if err != nil {
		return err
	}

	// 称号のtier変化を削除の前後で比較するため、削除前の時点で取得しておく。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, match.UserId)

	if err := u.repository.Delete(ctx, id); err != nil {
		return err
	}

	if tierErr == nil {
		u.designationEvaluation.NotifyIfTierLost(ctx, match.UserId, beforeTier)
	}

	return nil
}

func (u *Match) Reorder(
	ctx context.Context,
	recordId string,
	orders []*entity.MatchOrder,
) error {
	return u.repository.Reorder(ctx, recordId, orders)
}
