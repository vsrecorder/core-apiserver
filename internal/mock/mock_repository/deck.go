// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/domain/repository/deck.go
//
// Generated by this command:
//
//	mockgen -source=./internal/domain/repository/deck.go -destination=./internal/mock/mock_repository/deck.go
//

// Package mock_repository is a generated GoMock package.
package mock_repository

import (
	context "context"
	reflect "reflect"
	time "time"

	entity "github.com/vsrecorder/core-apiserver/internal/domain/entity"
	gomock "go.uber.org/mock/gomock"
)

// MockDeckInterface is a mock of DeckInterface interface.
type MockDeckInterface struct {
	ctrl     *gomock.Controller
	recorder *MockDeckInterfaceMockRecorder
	isgomock struct{}
}

// MockDeckInterfaceMockRecorder is the mock recorder for MockDeckInterface.
type MockDeckInterfaceMockRecorder struct {
	mock *MockDeckInterface
}

// NewMockDeckInterface creates a new mock instance.
func NewMockDeckInterface(ctrl *gomock.Controller) *MockDeckInterface {
	mock := &MockDeckInterface{ctrl: ctrl}
	mock.recorder = &MockDeckInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDeckInterface) EXPECT() *MockDeckInterfaceMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockDeckInterface) Delete(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockDeckInterfaceMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockDeckInterface)(nil).Delete), ctx, id)
}

// Find mocks base method.
func (m *MockDeckInterface) Find(ctx context.Context, limit, offset int) ([]*entity.Deck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Find", ctx, limit, offset)
	ret0, _ := ret[0].([]*entity.Deck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Find indicates an expected call of Find.
func (mr *MockDeckInterfaceMockRecorder) Find(ctx, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Find", reflect.TypeOf((*MockDeckInterface)(nil).Find), ctx, limit, offset)
}

// FindById mocks base method.
func (m *MockDeckInterface) FindById(ctx context.Context, id string) (*entity.Deck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindById", ctx, id)
	ret0, _ := ret[0].(*entity.Deck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindById indicates an expected call of FindById.
func (mr *MockDeckInterfaceMockRecorder) FindById(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindById", reflect.TypeOf((*MockDeckInterface)(nil).FindById), ctx, id)
}

// FindByUserId mocks base method.
func (m *MockDeckInterface) FindByUserId(ctx context.Context, uid string, archivedFlg bool, limit, offset int) ([]*entity.Deck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByUserId", ctx, uid, archivedFlg, limit, offset)
	ret0, _ := ret[0].([]*entity.Deck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByUserId indicates an expected call of FindByUserId.
func (mr *MockDeckInterfaceMockRecorder) FindByUserId(ctx, uid, archivedFlg, limit, offset any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByUserId", reflect.TypeOf((*MockDeckInterface)(nil).FindByUserId), ctx, uid, archivedFlg, limit, offset)
}

// FindByUserIdOnCursor mocks base method.
func (m *MockDeckInterface) FindByUserIdOnCursor(ctx context.Context, uid string, archivedFlg bool, limit int, cursor time.Time) ([]*entity.Deck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByUserIdOnCursor", ctx, uid, archivedFlg, limit, cursor)
	ret0, _ := ret[0].([]*entity.Deck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByUserIdOnCursor indicates an expected call of FindByUserIdOnCursor.
func (mr *MockDeckInterfaceMockRecorder) FindByUserIdOnCursor(ctx, uid, archivedFlg, limit, cursor any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByUserIdOnCursor", reflect.TypeOf((*MockDeckInterface)(nil).FindByUserIdOnCursor), ctx, uid, archivedFlg, limit, cursor)
}

// FindOnCursor mocks base method.
func (m *MockDeckInterface) FindOnCursor(ctx context.Context, limit int, cursor time.Time) ([]*entity.Deck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindOnCursor", ctx, limit, cursor)
	ret0, _ := ret[0].([]*entity.Deck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindOnCursor indicates an expected call of FindOnCursor.
func (mr *MockDeckInterfaceMockRecorder) FindOnCursor(ctx, limit, cursor any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindOnCursor", reflect.TypeOf((*MockDeckInterface)(nil).FindOnCursor), ctx, limit, cursor)
}

// Save mocks base method.
func (m *MockDeckInterface) Save(ctx context.Context, entity *entity.Deck) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save", ctx, entity)
	ret0, _ := ret[0].(error)
	return ret0
}

// Save indicates an expected call of Save.
func (mr *MockDeckInterfaceMockRecorder) Save(ctx, entity any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*MockDeckInterface)(nil).Save), ctx, entity)
}
