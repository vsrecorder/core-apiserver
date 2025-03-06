// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/usecase/official_event.go
//
// Generated by this command:
//
//	mockgen -source=./internal/usecase/official_event.go -destination=./internal/mock/mock_usecase/official_event.go
//

// Package mock_usecase is a generated GoMock package.
package mock_usecase

import (
	context "context"
	reflect "reflect"
	time "time"

	entity "github.com/vsrecorder/core-apiserver/internal/domain/entity"
	gomock "go.uber.org/mock/gomock"
)

// MockOfficialEventInterface is a mock of OfficialEventInterface interface.
type MockOfficialEventInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOfficialEventInterfaceMockRecorder
	isgomock struct{}
}

// MockOfficialEventInterfaceMockRecorder is the mock recorder for MockOfficialEventInterface.
type MockOfficialEventInterfaceMockRecorder struct {
	mock *MockOfficialEventInterface
}

// NewMockOfficialEventInterface creates a new mock instance.
func NewMockOfficialEventInterface(ctrl *gomock.Controller) *MockOfficialEventInterface {
	mock := &MockOfficialEventInterface{ctrl: ctrl}
	mock.recorder = &MockOfficialEventInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOfficialEventInterface) EXPECT() *MockOfficialEventInterfaceMockRecorder {
	return m.recorder
}

// Find mocks base method.
func (m *MockOfficialEventInterface) Find(ctx context.Context, typeId, leagueType uint, startDate, endDate time.Time) ([]*entity.OfficialEvent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Find", ctx, typeId, leagueType, startDate, endDate)
	ret0, _ := ret[0].([]*entity.OfficialEvent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Find indicates an expected call of Find.
func (mr *MockOfficialEventInterfaceMockRecorder) Find(ctx, typeId, leagueType, startDate, endDate any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Find", reflect.TypeOf((*MockOfficialEventInterface)(nil).Find), ctx, typeId, leagueType, startDate, endDate)
}

// FindById mocks base method.
func (m *MockOfficialEventInterface) FindById(ctx context.Context, id uint) (*entity.OfficialEvent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindById", ctx, id)
	ret0, _ := ret[0].(*entity.OfficialEvent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindById indicates an expected call of FindById.
func (mr *MockOfficialEventInterfaceMockRecorder) FindById(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindById", reflect.TypeOf((*MockOfficialEventInterface)(nil).FindById), ctx, id)
}
