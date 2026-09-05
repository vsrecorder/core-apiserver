// Package apperror はアプリケーション全体で共有するドメインエラーを定義する。
//
// 各層(infrastructure / usecase / controller)は、ここで定義された
// センチネルエラーを errors.Is で判定することで、特定のライブラリ実装
// (gorm など)や層をまたいだ重複定義に依存せず、エラーの種類を一意に
// 扱えるようにする。
package apperror

import "errors"

var (
	// ErrRecordNotFound は対象のリソースが存在しない場合に返す。
	// infrastructure 層で gorm.ErrRecordNotFound から変換され、
	// 上位層は gorm に依存せずこのエラーで判定する。
	// HTTP では 404 Not Found に対応する。
	ErrRecordNotFound = errors.New("record not found")

	// ErrAlreadyExists は作成しようとしたリソースが既に存在する場合に返す。
	// HTTP では 409 Conflict に対応する。
	ErrAlreadyExists = errors.New("already exists")

	// ErrWithdrawn は退会済み(論理削除済み)のユーザーに対する操作を拒否する場合に返す。
	// 「まだ存在しない(404)」ではなく「かつて存在したが失われた」ことを表すため、
	// HTTP では 410 Gone に対応する。
	ErrWithdrawn = errors.New("withdrawn")

	// ErrUnderMaintenance は依存する外部サイトがメンテナンス中で、
	// 処理を継続できない場合に返す。HTTP では 503 Service Unavailable に対応する。
	ErrUnderMaintenance = errors.New("under maintenance")

	// ErrDeckCodeInvalid はデッキコードが正しくない場合に返す。
	// HTTP では 400 Bad Request に対応する。
	ErrDeckCodeInvalid = errors.New("deck code is invalid")

	// ErrLocked は一定期間内の再変更が禁止されている場合に返す。
	// HTTP では 409 Conflict に対応する。
	ErrLocked = errors.New("locked")

	// ErrInvalidMatchOrder は match の並び替えリクエストに含まれるIDが、
	// 対象record内の未削除match集合と過不足なく一致しない場合に返す。
	// HTTP では 400 Bad Request に対応する。
	ErrInvalidMatchOrder = errors.New("invalid match order")

	// ErrInvalidMatch は対戦結果(BO3/両者引き分け含む)の整合性が取れていない場合に返す。
	// 例: BO3なのにゲーム数が不正、ゲームの勝敗と対戦全体の勝敗が食い違う 等。
	// HTTP では 400 Bad Request に対応する。
	ErrInvalidMatch = errors.New("invalid match")

	// ErrInvalidRecord は記録の整合性が取れていない場合に返す。
	// 例: 紐づくイベント種別(公式/Tonamel/フレンド/自由形式)がちょうど1つでない。
	// HTTP では 400 Bad Request に対応する。
	ErrInvalidRecord = errors.New("invalid record")

	// ErrNoKnownActivityCategory は日次活動の記録リクエストに、既知の計測カテゴリが
	// 1つも含まれていない場合に返す。未知のカテゴリが混ざっているだけでは返さない
	// (既知のぶんだけ記録する)。HTTP では 400 Bad Request に対応する。
	ErrNoKnownActivityCategory = errors.New("no known activity category")

	// ErrTooManyPushSubscriptions は1ユーザーの生きている push 購読が上限に達している場合に返す。
	// endpoint を変えて無制限に登録されると、配信バッチが外部へ POST を繰り返す増幅器になるため。
	// HTTP では 409 Conflict に対応する。
	ErrTooManyPushSubscriptions = errors.New("too many push subscriptions")

	// ErrTooManyUserGyms は1ユーザーのMyジム登録数が上限に達している場合に返す。
	// 上限を超えたぶんを黙って押し出さず、どれを外すかはユーザーに選ばせる。
	// HTTP では 409 Conflict に対応する。
	ErrTooManyUserGyms = errors.New("too many user gyms")

	// ErrDeckArchived はアーカイブ済みのデッキに対して、公開など
	// 利用中のデッキにしかできない操作をしようとした場合に返す。HTTP では 409 に対応する。
	ErrDeckArchived = errors.New("deck is archived")

	// ErrRepublishTooSoon は同じデッキコードを短い間隔で公開し直そうとした場合に返す
	// (みんなの公開デッキへの掲載)。HTTP では 429 に対応する。
	ErrRepublishTooSoon = errors.New("republish too soon")

	// ErrDeckCodePostHidden は運営が非表示にした投稿のデッキコードを公開し直そうとした場合に返す
	// (取り下げて別の投稿として作り直しても表示に戻せないようにする)。HTTP では 403 に対応する。
	ErrDeckCodePostHidden = errors.New("deck code post is hidden")
)
