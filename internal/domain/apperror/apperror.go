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

	// ErrUnderMaintenance は依存する外部サイトがメンテナンス中で、
	// 処理を継続できない場合に返す。HTTP では 503 Service Unavailable に対応する。
	ErrUnderMaintenance = errors.New("under maintenance")

	// ErrDeckCodeInvalid はデッキコードが正しくない場合に返す。
	// HTTP では 400 Bad Request に対応する。
	ErrDeckCodeInvalid = errors.New("deck code is invalid")

	// ErrLocked は一定期間内の再変更が禁止されている場合に返す。
	// HTTP では 409 Conflict に対応する。
	ErrLocked = errors.New("locked")

	// ErrInvalidChallenge は所有権確認用のチャレンジトークンが不正・期限切れ、
	// または発行時と異なるユーザー/対象に対して使われた場合に返す。
	// HTTP では 400 Bad Request に対応する。
	ErrInvalidChallenge = errors.New("invalid challenge")

	// ErrOwnershipNotVerified は所有権確認チャレンジ(アバター変更)がまだ
	// 完了していない場合に返す。HTTP では 403 Forbidden に対応する。
	ErrOwnershipNotVerified = errors.New("ownership not verified")

	// ErrInvalidMatchOrder は match の並び替えリクエストに含まれるIDが、
	// 対象record内の未削除match集合と過不足なく一致しない場合に返す。
	// HTTP では 400 Bad Request に対応する。
	ErrInvalidMatchOrder = errors.New("invalid match order")
)
