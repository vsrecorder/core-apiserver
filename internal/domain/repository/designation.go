package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type DesignationInterface interface {
	// FindAll は tier 昇順で全ての称号定義を返す。
	FindAll(
		ctx context.Context,
	) ([]*entity.Designation, error)
}
