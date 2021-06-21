// Code generated by MockGen. DO NOT EDIT.
// Source: net (interfaces: Listener,Error)

// Package mock_net is a generated GoMock package.
package mock_net

import (
	net "net"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockListener is a mock of Listener interface.
type MockListener struct {
	ctrl     *gomock.Controller
	recorder *MockListenerMockRecorder
}

// MockListenerMockRecorder is the mock recorder for MockListener.
type MockListenerMockRecorder struct {
	mock *MockListener
}

// NewMockListener creates a new mock instance.
func NewMockListener(ctrl *gomock.Controller) *MockListener {
	mock := &MockListener{ctrl: ctrl}
	mock.recorder = &MockListenerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockListener) EXPECT() *MockListenerMockRecorder {
	return m.recorder
}

// Accept mocks base method.
func (m *MockListener) Accept() (net.Conn, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Accept")
	ret0, _ := ret[0].(net.Conn)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Accept indicates an expected call of Accept.
func (mr *MockListenerMockRecorder) Accept() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Accept", reflect.TypeOf((*MockListener)(nil).Accept))
}

// Addr mocks base method.
func (m *MockListener) Addr() net.Addr {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Addr")
	ret0, _ := ret[0].(net.Addr)
	return ret0
}

// Addr indicates an expected call of Addr.
func (mr *MockListenerMockRecorder) Addr() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Addr", reflect.TypeOf((*MockListener)(nil).Addr))
}

// Close mocks base method.
func (m *MockListener) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockListenerMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockListener)(nil).Close))
}

// MockError is a mock of Error interface.
type MockError struct {
	ctrl     *gomock.Controller
	recorder *MockErrorMockRecorder
}

// MockErrorMockRecorder is the mock recorder for MockError.
type MockErrorMockRecorder struct {
	mock *MockError
}

// NewMockError creates a new mock instance.
func NewMockError(ctrl *gomock.Controller) *MockError {
	mock := &MockError{ctrl: ctrl}
	mock.recorder = &MockErrorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockError) EXPECT() *MockErrorMockRecorder {
	return m.recorder
}

// Error mocks base method.
func (m *MockError) Error() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Error")
	ret0, _ := ret[0].(string)
	return ret0
}

// Error indicates an expected call of Error.
func (mr *MockErrorMockRecorder) Error() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockError)(nil).Error))
}

// Temporary mocks base method.
func (m *MockError) Temporary() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Temporary")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Temporary indicates an expected call of Temporary.
func (mr *MockErrorMockRecorder) Temporary() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Temporary", reflect.TypeOf((*MockError)(nil).Temporary))
}

// Timeout mocks base method.
func (m *MockError) Timeout() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Timeout")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Timeout indicates an expected call of Timeout.
func (mr *MockErrorMockRecorder) Timeout() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Timeout", reflect.TypeOf((*MockError)(nil).Timeout))
}
