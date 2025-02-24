// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/domain/repository/match.go
//
// Generated by this command:
//
//	mockgen -source=./internal/domain/repository/match.go -destination=./internal/mock/mock_repository/match.go
//

// Package mock_repository is a generated GoMock package.
package mock_repository

import (
	context "context"
	reflect "reflect"

	entity "github.com/vsrecorder/core-apiserver/internal/domain/entity"
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
