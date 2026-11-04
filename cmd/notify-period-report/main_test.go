package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubEnvironments は resolveTargets に渡す環境の一覧。GORM と同じく DATE を UTC 0時で持つ。
type stubEnvironments []*entity.Environment

func (s stubEnvironments) Find(context.Context) ([]*entity.Environment, error) {
	return s, nil
}

func (s stubEnvironments) FindById(_ context.Context, id string) (*entity.Environment, error) {
	for _, env := range s {
		if env.ID == id {
			return env, nil
		}
	}
	return nil, errors.New("not found")
}

func utcDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func localDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

var testEnvironments = stubEnvironments{
	entity.NewEnvironment("m6a", "30th CELEBRATION", utcDate(2026, 9, 16), utcDate(2026, 11, 26)),
	entity.NewEnvironment("m6", "ストームエメラルダ", utcDate(2026, 7, 31), utcDate(2026, 9, 15)),
}

func TestResolveTargets(t *testing.T) {
	ctx := context.Background()

	t.Run("正常系_1日の昼に実行すると先月が対象になる", func(t *testing.T) {
		got, err := resolveTargets(ctx, testEnvironments, "", "", "", time.Date(2026, 11, 1, 12, 0, 0, 0, time.Local))

		require.NoError(t, err)
		require.Equal(t, []target{{kind: "month", key: "2026-10", from: localDate(2026, 10, 1), to: localDate(2026, 11, 1)}}, got)
	})

	t.Run("正常系_1月1日は前の年の12月が対象になる", func(t *testing.T) {
		got, err := resolveTargets(ctx, testEnvironments, "2027-01-01", "", "", time.Now())

		require.NoError(t, err)
		require.Equal(t, []target{{kind: "month", key: "2026-12", from: localDate(2026, 12, 1), to: localDate(2027, 1, 1)}}, got)
	})

	t.Run("正常系_前日に終わった環境があればその環境が対象になる", func(t *testing.T) {
		got, err := resolveTargets(ctx, testEnvironments, "2026-11-27", "", "", time.Now())

		require.NoError(t, err)
		require.Equal(t, []target{{kind: "environment", key: "m6a", from: localDate(2026, 9, 16), to: localDate(2026, 11, 27)}}, got)
	})

	t.Run("正常系_月初と環境の切り替わりが重なれば両方が対象になる", func(t *testing.T) {
		envs := stubEnvironments{
			entity.NewEnvironment("x1", "テスト環境", utcDate(2026, 9, 1), utcDate(2026, 11, 30)),
		}

		got, err := resolveTargets(ctx, envs, "2026-12-01", "", "", time.Now())

		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, "2026-11", got[0].key)
		require.Equal(t, "x1", got[1].key)
	})

	t.Run("正常系_月初でも環境の初日でもなければ何も対象にならない", func(t *testing.T) {
		// m6a の最終日当日(まだ終わっていない)
		got, err := resolveTargets(ctx, testEnvironments, "2026-11-26", "", "", time.Now())

		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("正常系_月や環境を直接指定すると実行日の判定は使わない", func(t *testing.T) {
		got, err := resolveTargets(ctx, testEnvironments, "2026-11-01", "2026-09", "m6", time.Now())

		require.NoError(t, err)
		require.Equal(t, []target{
			{kind: "month", key: "2026-09", from: localDate(2026, 9, 1), to: localDate(2026, 10, 1)},
			{kind: "environment", key: "m6", from: localDate(2026, 7, 31), to: localDate(2026, 9, 16)},
		}, got)
	})

	t.Run("異常系_不正な指定はエラーにする", func(t *testing.T) {
		_, err := resolveTargets(ctx, testEnvironments, "2026-13-01", "", "", time.Now())
		require.Error(t, err)

		_, err = resolveTargets(ctx, testEnvironments, "", "2026-13", "", time.Now())
		require.Error(t, err)

		_, err = resolveTargets(ctx, testEnvironments, "", "", "zzz", time.Now())
		require.Error(t, err)
	})
}
