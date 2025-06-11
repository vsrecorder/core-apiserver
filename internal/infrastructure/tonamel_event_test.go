package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTonamelEventInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindById": test_TonamelEventInfrastructure_FindById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_TonamelEventInfrastructure_FindById(t *testing.T) {
	r := NewTonamelEvent()

	id := "OakZc"
	title := "第23回 ACEカップ～FINAL～ - Tonamel"

	res, err := r.FindById(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, res.ID, id)
	require.Equal(t, res.Title, title)
	require.NotEmpty(t, res.Image)
}
