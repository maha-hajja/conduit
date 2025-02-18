// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/conduitio/conduit/pkg/web/api (interfaces: ConnectorOrchestrator)

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	connector "github.com/conduitio/conduit/pkg/connector"
	gomock "github.com/golang/mock/gomock"
)

// ConnectorOrchestrator is a mock of ConnectorOrchestrator interface.
type ConnectorOrchestrator struct {
	ctrl     *gomock.Controller
	recorder *ConnectorOrchestratorMockRecorder
}

// ConnectorOrchestratorMockRecorder is the mock recorder for ConnectorOrchestrator.
type ConnectorOrchestratorMockRecorder struct {
	mock *ConnectorOrchestrator
}

// NewConnectorOrchestrator creates a new mock instance.
func NewConnectorOrchestrator(ctrl *gomock.Controller) *ConnectorOrchestrator {
	mock := &ConnectorOrchestrator{ctrl: ctrl}
	mock.recorder = &ConnectorOrchestratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *ConnectorOrchestrator) EXPECT() *ConnectorOrchestratorMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *ConnectorOrchestrator) Create(arg0 context.Context, arg1 connector.Type, arg2 connector.Config) (connector.Connector, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2)
	ret0, _ := ret[0].(connector.Connector)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *ConnectorOrchestratorMockRecorder) Create(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*ConnectorOrchestrator)(nil).Create), arg0, arg1, arg2)
}

// Delete mocks base method.
func (m *ConnectorOrchestrator) Delete(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *ConnectorOrchestratorMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*ConnectorOrchestrator)(nil).Delete), arg0, arg1)
}

// Get mocks base method.
func (m *ConnectorOrchestrator) Get(arg0 context.Context, arg1 string) (connector.Connector, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(connector.Connector)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *ConnectorOrchestratorMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*ConnectorOrchestrator)(nil).Get), arg0, arg1)
}

// List mocks base method.
func (m *ConnectorOrchestrator) List(arg0 context.Context) map[string]connector.Connector {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0)
	ret0, _ := ret[0].(map[string]connector.Connector)
	return ret0
}

// List indicates an expected call of List.
func (mr *ConnectorOrchestratorMockRecorder) List(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*ConnectorOrchestrator)(nil).List), arg0)
}

// Update mocks base method.
func (m *ConnectorOrchestrator) Update(arg0 context.Context, arg1 string, arg2 connector.Config) (connector.Connector, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1, arg2)
	ret0, _ := ret[0].(connector.Connector)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *ConnectorOrchestratorMockRecorder) Update(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*ConnectorOrchestrator)(nil).Update), arg0, arg1, arg2)
}
