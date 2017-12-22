// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/urandom/readeef/content/repo (interfaces: Tag)

// Package mock_repo is a generated GoMock package.
package mock_repo

import (
	gomock "github.com/golang/mock/gomock"
	content "github.com/urandom/readeef/content"
	reflect "reflect"
)

// MockTag is a mock of Tag interface
type MockTag struct {
	ctrl     *gomock.Controller
	recorder *MockTagMockRecorder
}

// MockTagMockRecorder is the mock recorder for MockTag
type MockTagMockRecorder struct {
	mock *MockTag
}

// NewMockTag creates a new mock instance
func NewMockTag(ctrl *gomock.Controller) *MockTag {
	mock := &MockTag{ctrl: ctrl}
	mock.recorder = &MockTagMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTag) EXPECT() *MockTagMockRecorder {
	return m.recorder
}

// FeedIDs mocks base method
func (m *MockTag) FeedIDs(arg0 content.Tag, arg1 content.User) ([]content.FeedID, error) {
	ret := m.ctrl.Call(m, "FeedIDs", arg0, arg1)
	ret0, _ := ret[0].([]content.FeedID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FeedIDs indicates an expected call of FeedIDs
func (mr *MockTagMockRecorder) FeedIDs(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FeedIDs", reflect.TypeOf((*MockTag)(nil).FeedIDs), arg0, arg1)
}

// ForFeed mocks base method
func (m *MockTag) ForFeed(arg0 content.Feed, arg1 content.User) ([]content.Tag, error) {
	ret := m.ctrl.Call(m, "ForFeed", arg0, arg1)
	ret0, _ := ret[0].([]content.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForFeed indicates an expected call of ForFeed
func (mr *MockTagMockRecorder) ForFeed(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForFeed", reflect.TypeOf((*MockTag)(nil).ForFeed), arg0, arg1)
}

// ForUser mocks base method
func (m *MockTag) ForUser(arg0 content.User) ([]content.Tag, error) {
	ret := m.ctrl.Call(m, "ForUser", arg0)
	ret0, _ := ret[0].([]content.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForUser indicates an expected call of ForUser
func (mr *MockTagMockRecorder) ForUser(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForUser", reflect.TypeOf((*MockTag)(nil).ForUser), arg0)
}

// Get mocks base method
func (m *MockTag) Get(arg0 content.TagID, arg1 content.User) (content.Tag, error) {
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(content.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockTagMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockTag)(nil).Get), arg0, arg1)
}