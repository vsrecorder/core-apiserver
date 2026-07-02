package repository

import "context"

// TransactionManager は複数のリポジトリ呼び出しを1つのDBトランザクションに
// まとめるための抽象。ctx に紐づけられたトランザクションを fn に伝播させることで、
// 各リポジトリの既存メソッドシグネチャを変更せずにトランザクションを合成できる。
type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
