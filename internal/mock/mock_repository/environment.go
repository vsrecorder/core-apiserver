// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/domain/repository/environment.go
//
// Generated by this command:
//
//	mockgen -source=./internal/domain/repository/environment.go -destination=./internal/mock/mock_repository/environment.go
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

// MockEnvironmentInterface is a mock of EnvironmentInterface interface.
type MockEnvironmentInterface struct {
	ctrl     *gomock.Controller
	recorder *MockEnvironmentInterfaceMockRecorder
	isgomock struct{}
}

// MockEnvironmentInterfaceMockRecorder is the mock recorder for MockEnvironmentInterface.
type MockEnvironmentInterfaceMockRecorder struct {
	mock *MockEnvironmentInterface
}

// NewMockEnvironmentInterface creates a new mock instance.
func NewMockEnvironmentInterface(ctrl *gomock.Controller) *MockEnvironmentInterface {
	mock := &MockEnvironmentInterface{ctrl: ctrl}
	mock.recorder = &MockEnvironmentInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnvironmentInterface) EXPECT() *MockEnvironmentInterfaceMockRecorder {
	return m.recorder
}

// Find mocks base method.
func (m *MockEnvironmentInterface) Find(ctx context.Context) ([]*entity.Environment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Find", ctx)
	ret0, _ := ret[0].([]*entity.Environment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Find indicates an expected call of Find.
func (mr *MockEnvironmentInterfaceMockRecorder) Find(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Find", reflect.TypeOf((*MockEnvironmentInterface)(nil).Find), ctx)
}

// FindByDate mocks base method.
func (m *MockEnvironmentInterface) FindByDate(ctx context.Context, date time.Time) (*entity.Environment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByDate", ctx, date)
	ret0, _ := ret[0].(*entity.Environment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByDate indicates an expected call of FindByDate.
func (mr *MockEnvironmentInterfaceMockRecorder) FindByDate(ctx, date any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByDate", reflect.TypeOf((*MockEnvironmentInterface)(nil).FindByDate), ctx, date)
}

// FindById mocks base method.
func (m *MockEnvironmentInterface) FindById(ctx context.Context, id string) (*entity.Environment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindById", ctx, id)
	ret0, _ := ret[0].(*entity.Environment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindById indicates an expected call of FindById.
func (mr *MockEnvironmentInterfaceMockRecorder) FindById(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindById", reflect.TypeOf((*MockEnvironmentInterface)(nil).FindById), ctx, id)
}

// FindByTerm mocks base method.
func (m *MockEnvironmentInterface) FindByTerm(ctx context.Context, fromDate, toDate time.Time) ([]*entity.Environment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByTerm", ctx, fromDate, toDate)
	ret0, _ := ret[0].([]*entity.Environment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByTerm indicates an expected call of FindByTerm.
func (mr *MockEnvironmentInterfaceMockRecorder) FindByTerm(ctx, fromDate, toDate any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByTerm", reflect.TypeOf((*MockEnvironmentInterface)(nil).FindByTerm), ctx, fromDate, toDate)
}
