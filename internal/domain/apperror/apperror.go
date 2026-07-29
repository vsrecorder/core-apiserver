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

	// ErrInvalidVerification は webapp が発行した検証済みトークンが不正・期限切れ、
	// または発行時と異なるユーザー/対象に対して使われた場合に返す。
	//
	// プレイヤーIDの所有権確認(実在確認とアバター変更の確認)は webapp が行い、
	// このAPIサーバはその結果を署名で検証するだけである点に注意。
	// HTTP では 400 Bad Request に対応する。
	ErrInvalidVerification = errors.New("invalid verification")

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
)
