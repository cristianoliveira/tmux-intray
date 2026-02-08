package search

import (
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/stretchr/testify/mock"
)

// MockProvider is a mock implementation of Provider for testing.
type MockProvider struct {
	mock.Mock
}

// Match provides a mock function with given fields: notif, query.
func (_m *MockProvider) Match(notif notification.Notification, query string) bool {
	_va := make([]interface{}, 2)
	_va[0] = notif
	_va[1] = query
	ret := _m.Called(_va...)

	var r0 bool
	if rf, ok := ret.Get(0).(func(notification.Notification, string) bool); ok {
		r0 = rf(notif, query)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Name provides a mock function with given fields: .
func (_m *MockProvider) Name() string {
	_va := make([]interface{}, 0)
	ret := _m.Called(_va...)

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
