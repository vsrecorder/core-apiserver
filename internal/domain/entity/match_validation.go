package entity

// MatchResultInput は対戦結果の整合性検証に必要な最小の入力。
// 先攻/後攻はサーバ側の整合検証には不要なため、ゲームは勝敗フラグ(WinningFlg)のみを持つ。
type MatchResultInput struct {
	BO3Flg               bool
	GroupMatchFlg        bool
	DefaultVictoryFlg    bool
	DefaultDefeatFlg     bool
	VictoryFlg           bool
	DrawFlg              bool
	GroupMatchVictoryFlg bool
	// GameWinningFlgs は各ゲームの勝敗(true=勝ち)。順番は 1本目→N本目。
	GameWinningFlgs []bool
}

// IsValidMatchResult は対戦結果(BO3/両者引き分けを含む)の整合性を検証する。
//
// フラグ同士の矛盾・ゲーム数・「各ゲームの勝敗」と「対戦全体の勝敗」の一致を確認する。
// RecordId の有無や入力欄の長さといった入力サニタイズは含まない(呼び出し側で行う)。
//
// controller(middleware)と usecase の両方から呼び、検証の二重実装による
// 仕様の乖離を防ぐための単一の真実。
func IsValidMatchResult(in MatchResultInput) bool {
	games := in.GameWinningFlgs

	// 不戦勝と不戦敗の両方が立っている
	if in.DefaultVictoryFlg && in.DefaultDefeatFlg {
		return false
	}

	// 不戦勝なのに勝ちでない / 不戦敗なのに勝ちになっている
	if in.DefaultVictoryFlg && !in.VictoryFlg {
		return false
	}
	if in.DefaultDefeatFlg && in.VictoryFlg {
		return false
	}

	// 不戦勝/不戦敗は対戦が行われていないため、Gameが存在してはならない
	isDefault := in.DefaultVictoryFlg || in.DefaultDefeatFlg
	if isDefault && len(games) > 0 {
		return false
	}

	// チームの勝敗(GroupMatchVictoryFlg)を持てるのはチーム戦のBO1のみ
	if in.GroupMatchVictoryFlg && (in.BO3Flg || !in.GroupMatchFlg) {
		return false
	}

	// 両者引き分け(ダブルドロー)を設定できるのはBO3のみ
	if in.DrawFlg && !in.BO3Flg {
		return false
	}

	// 引き分けは勝ち・不戦勝/不戦敗と両立しない
	if in.DrawFlg && in.VictoryFlg {
		return false
	}
	if in.DrawFlg && isDefault {
		return false
	}

	// 不戦勝/不戦敗の場合はGameが存在しないため、ここから先の検証は行わない
	if isDefault {
		return true
	}

	switch {
	case in.BO3Flg && in.DrawFlg:
		// 両者引き分け: 1勝1敗のまま時間切れで決着したケース。2ゲーム(1勝1敗)のみ。
		if len(games) != 2 {
			return false
		}
		if games[0] == games[1] {
			return false
		}
	case in.BO3Flg:
		// BO3(2本先取)は2ゲーム(2-0/0-2)または3ゲーム(2-1/1-2)で決着する
		if len(games) != 2 && len(games) != 3 {
			return false
		}
		if len(games) == 2 {
			// 2連勝(2-0)か2連敗(0-2)。どちらのゲームも対戦全体の勝敗と一致する
			if games[0] != in.VictoryFlg || games[1] != in.VictoryFlg {
				return false
			}
		}
		if len(games) == 3 {
			// 3ゲーム目が行われるのは1勝1敗のときのみ
			if games[0] == games[1] {
				return false
			}
			// 3ゲーム目の勝敗が対戦全体の勝敗になる
			if games[2] != in.VictoryFlg {
				return false
			}
		}
	default:
		// BO1(1本勝負)はちょうど1ゲームで決着し、その勝敗が対戦全体の勝敗になる
		if len(games) != 1 {
			return false
		}
		if games[0] != in.VictoryFlg {
			return false
		}
	}

	return true
}
