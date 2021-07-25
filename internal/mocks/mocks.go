// Code generated by MockGen. DO NOT EDIT.
// Source: internal/interface.go

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	http "net/http"
	reflect "reflect"
)

// MockArtifact is a mock of Artifact interface
type MockArtifact struct {
	ctrl     *gomock.Controller
	recorder *MockArtifactMockRecorder
}

// MockArtifactMockRecorder is the mock recorder for MockArtifact
type MockArtifactMockRecorder struct {
	mock *MockArtifact
}

// NewMockArtifact creates a new mock instance
func NewMockArtifact(ctrl *gomock.Controller) *MockArtifact {
	mock := &MockArtifact{ctrl: ctrl}
	mock.recorder = &MockArtifactMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockArtifact) EXPECT() *MockArtifactMockRecorder {
	return m.recorder
}

// TryMakeRequest mocks base method
func (m *MockArtifact) TryMakeRequest(host string, w http.ResponseWriter, r *http.Request, token string, responseHandler func(*http.Response) bool) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TryMakeRequest", host, w, r, token, responseHandler)
	ret0, _ := ret[0].(bool)
	return ret0
}

// TryMakeRequest indicates an expected call of TryMakeRequest
func (mr *MockArtifactMockRecorder) TryMakeRequest(host, w, r, token, responseHandler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TryMakeRequest", reflect.TypeOf((*MockArtifact)(nil).TryMakeRequest), host, w, r, token, responseHandler)
}

// MockAuth is a mock of Auth interface
type MockAuth struct {
	ctrl     *gomock.Controller
	recorder *MockAuthMockRecorder
}

// MockAuthMockRecorder is the mock recorder for MockAuth
type MockAuthMockRecorder struct {
	mock *MockAuth
}

// NewMockAuth creates a new mock instance
func NewMockAuth(ctrl *gomock.Controller) *MockAuth {
	mock := &MockAuth{ctrl: ctrl}
	mock.recorder = &MockAuthMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAuth) EXPECT() *MockAuthMockRecorder {
	return m.recorder
}

// IsAuthSupported mocks base method
func (m *MockAuth) IsAuthSupported() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAuthSupported")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsAuthSupported indicates an expected call of IsAuthSupported
func (mr *MockAuthMockRecorder) IsAuthSupported() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAuthSupported", reflect.TypeOf((*MockAuth)(nil).IsAuthSupported))
}

// RequireAuth mocks base method
func (m *MockAuth) RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequireAuth", w, r)
	ret0, _ := ret[0].(bool)
	return ret0
}

// RequireAuth indicates an expected call of RequireAuth
func (mr *MockAuthMockRecorder) RequireAuth(w, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequireAuth", reflect.TypeOf((*MockAuth)(nil).RequireAuth), w, r)
}

// GetTokenIfExists mocks base method
func (m *MockAuth) GetTokenIfExists(w http.ResponseWriter, r *http.Request) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTokenIfExists", w, r)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTokenIfExists indicates an expected call of GetTokenIfExists
func (mr *MockAuthMockRecorder) GetTokenIfExists(w, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTokenIfExists", reflect.TypeOf((*MockAuth)(nil).GetTokenIfExists), w, r)
}

// CheckResponseForInvalidToken mocks base method
func (m *MockAuth) CheckResponseForInvalidToken(w http.ResponseWriter, r *http.Request, resp *http.Response) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckResponseForInvalidToken", w, r, resp)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CheckResponseForInvalidToken indicates an expected call of CheckResponseForInvalidToken
func (mr *MockAuthMockRecorder) CheckResponseForInvalidToken(w, r, resp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckResponseForInvalidToken", reflect.TypeOf((*MockAuth)(nil).CheckResponseForInvalidToken), w, r, resp)
}
