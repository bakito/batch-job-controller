// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/bakito/batch-job-controller/pkg/lifecycle (interfaces: Controller)
//
// Generated by this command:
//
//	mockgen -destination pkg/mocks/lifecycle/mock.go github.com/bakito/batch-job-controller/pkg/lifecycle Controller
//

// Package mock_lifecycle is a generated GoMock package.
package mock_lifecycle

import (
	reflect "reflect"

	config "github.com/bakito/batch-job-controller/pkg/config"
	lifecycle "github.com/bakito/batch-job-controller/pkg/lifecycle"
	metrics "github.com/bakito/batch-job-controller/pkg/metrics"
	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
)

// MockController is a mock of Controller interface.
type MockController struct {
	ctrl     *gomock.Controller
	recorder *MockControllerMockRecorder
}

// MockControllerMockRecorder is the mock recorder for MockController.
type MockControllerMockRecorder struct {
	mock *MockController
}

// NewMockController creates a new mock instance.
func NewMockController(ctrl *gomock.Controller) *MockController {
	mock := &MockController{ctrl: ctrl}
	mock.recorder = &MockControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockController) EXPECT() *MockControllerMockRecorder {
	return m.recorder
}

// AddPod mocks base method.
func (m *MockController) AddPod(arg0 lifecycle.Job) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddPod", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddPod indicates an expected call of AddPod.
func (mr *MockControllerMockRecorder) AddPod(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddPod", reflect.TypeOf((*MockController)(nil).AddPod), arg0)
}

// AllAdded mocks base method.
func (m *MockController) AllAdded(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllAdded", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// AllAdded indicates an expected call of AllAdded.
func (mr *MockControllerMockRecorder) AllAdded(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllAdded", reflect.TypeOf((*MockController)(nil).AllAdded), arg0)
}

// Config mocks base method.
func (m *MockController) Config() config.Config {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Config")
	ret0, _ := ret[0].(config.Config)
	return ret0
}

// Config indicates an expected call of Config.
func (mr *MockControllerMockRecorder) Config() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Config", reflect.TypeOf((*MockController)(nil).Config))
}

// Has mocks base method.
func (m *MockController) Has(arg0, arg1 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Has", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Has indicates an expected call of Has.
func (mr *MockControllerMockRecorder) Has(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Has", reflect.TypeOf((*MockController)(nil).Has), arg0, arg1)
}

// NewExecution mocks base method.
func (m *MockController) NewExecution(arg0 int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewExecution", arg0)
	ret0, _ := ret[0].(string)
	return ret0
}

// NewExecution indicates an expected call of NewExecution.
func (mr *MockControllerMockRecorder) NewExecution(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewExecution", reflect.TypeOf((*MockController)(nil).NewExecution), arg0)
}

// PodTerminated mocks base method.
func (m *MockController) PodTerminated(arg0, arg1 string, arg2 v1.PodPhase) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PodTerminated", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// PodTerminated indicates an expected call of PodTerminated.
func (mr *MockControllerMockRecorder) PodTerminated(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PodTerminated", reflect.TypeOf((*MockController)(nil).PodTerminated), arg0, arg1, arg2)
}

// ReportReceived mocks base method.
func (m *MockController) ReportReceived(arg0, arg1 string, arg2 error, arg3 metrics.Results) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ReportReceived", arg0, arg1, arg2, arg3)
}

// ReportReceived indicates an expected call of ReportReceived.
func (mr *MockControllerMockRecorder) ReportReceived(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportReceived", reflect.TypeOf((*MockController)(nil).ReportReceived), arg0, arg1, arg2, arg3)
}
