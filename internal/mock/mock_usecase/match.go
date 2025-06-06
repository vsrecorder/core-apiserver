// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/usecase/match.go
//
// Generated by this command:
//
//	mockgen -source=./internal/usecase/match.go -destination=./internal/mock/mock_usecase/match.go
//

// Package mock_usecase is a generated GoMock package.
package mock_usecase

import (
	context "context"
	reflect "reflect"

	entity "github.com/vsrecorder/core-apiserver/internal/domain/entity"
	usecase "github.com/vsrecorder/core-apiserver/internal/usecase"
	gomock "go.uber.org/mock/gomock"
)

// MockMatchInterface is a mock of MatchInterface interface.
type MockMatchInterface struct {
	ctrl     *gomock.Controller
	recorder *MockMatchInterfaceMockRecorder
	isgomock struct{}
}

// MockMatchInterfaceMockRecorder is the mock recorder for MockMatchInterface.
type MockMatchInterfaceMockRecorder struct {
	mock *MockMatchInterface
}

// NewMockMatchInterface creates a new mock instance.
func NewMockMatchInterface(ctrl *gomock.Controller) *MockMatchInterface {
	mock := &MockMatchInterface{ctrl: ctrl}
	mock.recorder = &MockMatchInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMatchInterface) EXPECT() *MockMatchInterfaceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockMatchInterface) Create(ctx context.Context, param *usecase.MatchParam) (*entity.Match, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, param)
	ret0, _ := ret[0].(*entity.Match)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockMatchInterfaceMockRecorder) Create(ctx, param any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockMatchInterface)(nil).Create), ctx, param)
}

// Delete mocks base method.
func (m *MockMatchInterface) Delete(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockMatchInterfaceMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockMatchInterface)(nil).Delete), ctx, id)
}

// FindById mocks base method.
func (m *MockMatchInterface) FindById(ctx context.Context, id string) (*entity.Match, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindById", ctx, id)
	ret0, _ := ret[0].(*entity.Match)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindById indicates an expected call of FindById.
func (mr *MockMatchInterfaceMockRecorder) FindById(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindById", reflect.TypeOf((*MockMatchInterface)(nil).FindById), ctx, id)
}

// FindByRecordId mocks base method.
func (m *MockMatchInterface) FindByRecordId(ctx context.Context, recordId string) ([]*entity.Match, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByRecordId", ctx, recordId)
	ret0, _ := ret[0].([]*entity.Match)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByRecordId indicates an expected call of FindByRecordId.
func (mr *MockMatchInterfaceMockRecorder) FindByRecordId(ctx, recordId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByRecordId", reflect.TypeOf((*MockMatchInterface)(nil).FindByRecordId), ctx, recordId)
}

// Update mocks base method.
func (m *MockMatchInterface) Update(ctx context.Context, id string, param *usecase.MatchParam) (*entity.Match, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, id, param)
	ret0, _ := ret[0].(*entity.Match)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockMatchInterfaceMockRecorder) Update(ctx, id, param any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockMatchInterface)(nil).Update), ctx, id, param)
}
