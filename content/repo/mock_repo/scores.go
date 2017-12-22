// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/urandom/readeef/content/repo (interfaces: Scores)

// Package mock_repo is a generated GoMock package.
package mock_repo

import (
	gomock "github.com/golang/mock/gomock"
	content "github.com/urandom/readeef/content"
	reflect "reflect"
)

// MockScores is a mock of Scores interface
type MockScores struct {
	ctrl     *gomock.Controller
	recorder *MockScoresMockRecorder
}

// MockScoresMockRecorder is the mock recorder for MockScores
type MockScoresMockRecorder struct {
	mock *MockScores
}

// NewMockScores creates a new mock instance
func NewMockScores(ctrl *gomock.Controller) *MockScores {
	mock := &MockScores{ctrl: ctrl}
	mock.recorder = &MockScoresMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockScores) EXPECT() *MockScoresMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockScores) Get(arg0 content.Article) (content.Scores, error) {
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(content.Scores)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockScoresMockRecorder) Get(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockScores)(nil).Get), arg0)
}

// Update mocks base method
func (m *MockScores) Update(arg0 content.Scores) error {
	ret := m.ctrl.Call(m, "Update", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockScoresMockRecorder) Update(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockScores)(nil).Update), arg0)
}