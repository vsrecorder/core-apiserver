// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/domain/repository/game.go
//
// Generated by this command:
//
//	mockgen -source=./internal/domain/repository/game.go -destination=./internal/mock/mock_repository/game.go
//

// Package mock_repository is a generated GoMock package.
package mock_repository

import (
	context "context"
	reflect "reflect"

	entity "github.com/vsrecorder/core-apiserver/internal/domain/entity"
	gomock "go.uber.org/mock/gomock"
)

// MockGameInterface is a mock of GameInterface interface.
type MockGameInterface struct {
	ctrl     *gomock.Controller
	recorder *MockGameInterfaceMockRecorder
	isgomock struct{}
}

// MockGameInterfaceMockRecorder is the mock recorder for MockGameInterface.
type MockGameInterfaceMockRecorder struct {
	mock *MockGameInterface
}

// NewMockGameInterface creates a new mock instance.
func NewMockGameInterface(ctrl *gomock.Controller) *MockGameInterface {
	mock := &MockGameInterface{ctrl: ctrl}
	mock.recorder = &MockGameInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGameInterface) EXPECT() *MockGameInterfaceMockRecorder {
	return m.recorder
}

// FindById mocks base method.
func (m *MockGameInterface) FindById(ctx context.Context, id string) (*entity.Game, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindById", ctx, id)
	ret0, _ := ret[0].(*entity.Game)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindById indicates an expected call of FindById.
func (mr *MockGameInterfaceMockRecorder) FindById(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindById", reflect.TypeOf((*MockGameInterface)(nil).FindById), ctx, id)
}

// FindByMatchId mocks base method.
func (m *MockGameInterface) FindByMatchId(ctx context.Context, matchId string) ([]*entity.Game, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByMatchId", ctx, matchId)
	ret0, _ := ret[0].([]*entity.Game)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByMatchId indicates an expected call of FindByMatchId.
func (mr *MockGameInterfaceMockRecorder) FindByMatchId(ctx, matchId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByMatchId", reflect.TypeOf((*MockGameInterface)(nil).FindByMatchId), ctx, matchId)
}
