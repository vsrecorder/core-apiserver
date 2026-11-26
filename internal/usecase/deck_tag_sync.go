package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// デッキのタグと、最新バージョンのタグを揃えるための処理。
//
// デッキのタグは「いまのデッキ」を表し、デッキ一覧やデッキ情報に出る。スプライトと同じく
// (deck_sprite_sync.go)、放っておくと「最新バージョンで ACE SPEC を入れ替えたのにデッキには
// 前のタグが残る」といった食い違いが起きる。そこで書き込みのたびに双方向で揃える:
//
//   - 最新バージョンのタグを決めた(作成・最新の編集・最新の削除で繰り上がった) → デッキへ写す
//   - デッキのタグを変えた → 最新バージョンへ写す
//
// スプライトとの違い:
//
//   - 空(タグなし)でも写す。スプライトの空は「未設定でデッキのものを表示する」だが、バージョンの
//     タグはデッキのタグで代わりに表示されることがないので、空は「タグを付けない」という指定そのもの。
//   - 過去のバージョンへの書き残しは要らない。上と同じ理由で、デッキのタグを変えても過去の
//     バージョンの表示は変わらない。

// tagIdsOf は付与済みのタグをID列にする(並びはそのまま)。戻り値は常に非 nil。
func tagIdsOf(tags []*entity.Tag) []string {
	ids := make([]string, 0, len(tags))
	for _, tag := range tags {
		ids = append(ids, tag.ID)
	}

	return ids
}

// syncDeckTagsWithLatestVersion は、最新バージョンのタグが tags に決まったときに、
// デッキのタグを同じものへ置き換える。トランザクションの中から呼ぶこと。
//
// tags は最新バージョンに実際に付いているタグ(付与できないものを除き、付与順に並べたもの)。
func syncDeckTagsWithLatestVersion(
	ctx context.Context,
	tagRepository repository.TagInterface,
	deckId string,
	tags []*entity.Tag,
) error {
	return tagRepository.ReplaceDeckTags(ctx, deckId, tagIdsOf(tags))
}

// syncLatestVersionTagsWithDeck は、デッキのタグが tags に決まったときに、
// 最新バージョンのタグを同じものへ置き換える。トランザクションの中から呼ぶこと。
//
// latest はデッキの最新バージョン。バージョンが無い(nil・ID が空)なら何もしない。
// 置き換えたら latest.Tags も揃える(デッキのレスポンスの latest_deck_code に載るため)。
func syncLatestVersionTagsWithDeck(
	ctx context.Context,
	tagRepository repository.TagInterface,
	latest *entity.DeckCode,
	tags []*entity.Tag,
) error {
	if latest == nil || latest.ID == "" {
		return nil
	}

	if err := tagRepository.ReplaceDeckCodeTags(ctx, latest.ID, tagIdsOf(tags)); err != nil {
		return err
	}
	latest.Tags = tags

	return nil
}
