// Code generated by mockery. DO NOT EDIT.

package biz

import (
	mock "github.com/stretchr/testify/mock"
	biz "github.com/tnborg/panel/internal/biz"

	request "github.com/tnborg/panel/internal/http/request"
)

// SettingRepo is an autogenerated mock type for the SettingRepo type
type SettingRepo struct {
	mock.Mock
}

type SettingRepo_Expecter struct {
	mock *mock.Mock
}

func (_m *SettingRepo) EXPECT() *SettingRepo_Expecter {
	return &SettingRepo_Expecter{mock: &_m.Mock}
}

// Delete provides a mock function with given fields: key
func (_m *SettingRepo) Delete(key biz.SettingKey) error {
	ret := _m.Called(key)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(biz.SettingKey) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SettingRepo_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type SettingRepo_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - key biz.SettingKey
func (_e *SettingRepo_Expecter) Delete(key interface{}) *SettingRepo_Delete_Call {
	return &SettingRepo_Delete_Call{Call: _e.mock.On("Delete", key)}
}

func (_c *SettingRepo_Delete_Call) Run(run func(key biz.SettingKey)) *SettingRepo_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(biz.SettingKey))
	})
	return _c
}

func (_c *SettingRepo_Delete_Call) Return(_a0 error) *SettingRepo_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SettingRepo_Delete_Call) RunAndReturn(run func(biz.SettingKey) error) *SettingRepo_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: key, defaultValue
func (_m *SettingRepo) Get(key biz.SettingKey, defaultValue ...string) (string, error) {
	_va := make([]interface{}, len(defaultValue))
	for _i := range defaultValue {
		_va[_i] = defaultValue[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...string) (string, error)); ok {
		return rf(key, defaultValue...)
	}
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...string) string); ok {
		r0 = rf(key, defaultValue...)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(biz.SettingKey, ...string) error); ok {
		r1 = rf(key, defaultValue...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SettingRepo_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type SettingRepo_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - key biz.SettingKey
//   - defaultValue ...string
func (_e *SettingRepo_Expecter) Get(key interface{}, defaultValue ...interface{}) *SettingRepo_Get_Call {
	return &SettingRepo_Get_Call{Call: _e.mock.On("Get",
		append([]interface{}{key}, defaultValue...)...)}
}

func (_c *SettingRepo_Get_Call) Run(run func(key biz.SettingKey, defaultValue ...string)) *SettingRepo_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]string, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(string)
			}
		}
		run(args[0].(biz.SettingKey), variadicArgs...)
	})
	return _c
}

func (_c *SettingRepo_Get_Call) Return(_a0 string, _a1 error) *SettingRepo_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SettingRepo_Get_Call) RunAndReturn(run func(biz.SettingKey, ...string) (string, error)) *SettingRepo_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetBool provides a mock function with given fields: key, defaultValue
func (_m *SettingRepo) GetBool(key biz.SettingKey, defaultValue ...bool) (bool, error) {
	_va := make([]interface{}, len(defaultValue))
	for _i := range defaultValue {
		_va[_i] = defaultValue[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetBool")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...bool) (bool, error)); ok {
		return rf(key, defaultValue...)
	}
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...bool) bool); ok {
		r0 = rf(key, defaultValue...)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(biz.SettingKey, ...bool) error); ok {
		r1 = rf(key, defaultValue...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SettingRepo_GetBool_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBool'
type SettingRepo_GetBool_Call struct {
	*mock.Call
}

// GetBool is a helper method to define mock.On call
//   - key biz.SettingKey
//   - defaultValue ...bool
func (_e *SettingRepo_Expecter) GetBool(key interface{}, defaultValue ...interface{}) *SettingRepo_GetBool_Call {
	return &SettingRepo_GetBool_Call{Call: _e.mock.On("GetBool",
		append([]interface{}{key}, defaultValue...)...)}
}

func (_c *SettingRepo_GetBool_Call) Run(run func(key biz.SettingKey, defaultValue ...bool)) *SettingRepo_GetBool_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]bool, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(bool)
			}
		}
		run(args[0].(biz.SettingKey), variadicArgs...)
	})
	return _c
}

func (_c *SettingRepo_GetBool_Call) Return(_a0 bool, _a1 error) *SettingRepo_GetBool_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SettingRepo_GetBool_Call) RunAndReturn(run func(biz.SettingKey, ...bool) (bool, error)) *SettingRepo_GetBool_Call {
	_c.Call.Return(run)
	return _c
}

// GetInt provides a mock function with given fields: key, defaultValue
func (_m *SettingRepo) GetInt(key biz.SettingKey, defaultValue ...int) (int, error) {
	_va := make([]interface{}, len(defaultValue))
	for _i := range defaultValue {
		_va[_i] = defaultValue[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetInt")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...int) (int, error)); ok {
		return rf(key, defaultValue...)
	}
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...int) int); ok {
		r0 = rf(key, defaultValue...)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(biz.SettingKey, ...int) error); ok {
		r1 = rf(key, defaultValue...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SettingRepo_GetInt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetInt'
type SettingRepo_GetInt_Call struct {
	*mock.Call
}

// GetInt is a helper method to define mock.On call
//   - key biz.SettingKey
//   - defaultValue ...int
func (_e *SettingRepo_Expecter) GetInt(key interface{}, defaultValue ...interface{}) *SettingRepo_GetInt_Call {
	return &SettingRepo_GetInt_Call{Call: _e.mock.On("GetInt",
		append([]interface{}{key}, defaultValue...)...)}
}

func (_c *SettingRepo_GetInt_Call) Run(run func(key biz.SettingKey, defaultValue ...int)) *SettingRepo_GetInt_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]int, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(int)
			}
		}
		run(args[0].(biz.SettingKey), variadicArgs...)
	})
	return _c
}

func (_c *SettingRepo_GetInt_Call) Return(_a0 int, _a1 error) *SettingRepo_GetInt_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SettingRepo_GetInt_Call) RunAndReturn(run func(biz.SettingKey, ...int) (int, error)) *SettingRepo_GetInt_Call {
	_c.Call.Return(run)
	return _c
}

// GetPanel provides a mock function with no fields
func (_m *SettingRepo) GetPanel() (*request.SettingPanel, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetPanel")
	}

	var r0 *request.SettingPanel
	var r1 error
	if rf, ok := ret.Get(0).(func() (*request.SettingPanel, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *request.SettingPanel); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.SettingPanel)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SettingRepo_GetPanel_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetPanel'
type SettingRepo_GetPanel_Call struct {
	*mock.Call
}

// GetPanel is a helper method to define mock.On call
func (_e *SettingRepo_Expecter) GetPanel() *SettingRepo_GetPanel_Call {
	return &SettingRepo_GetPanel_Call{Call: _e.mock.On("GetPanel")}
}

func (_c *SettingRepo_GetPanel_Call) Run(run func()) *SettingRepo_GetPanel_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *SettingRepo_GetPanel_Call) Return(_a0 *request.SettingPanel, _a1 error) *SettingRepo_GetPanel_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SettingRepo_GetPanel_Call) RunAndReturn(run func() (*request.SettingPanel, error)) *SettingRepo_GetPanel_Call {
	_c.Call.Return(run)
	return _c
}

// GetSlice provides a mock function with given fields: key, defaultValue
func (_m *SettingRepo) GetSlice(key biz.SettingKey, defaultValue ...[]string) ([]string, error) {
	_va := make([]interface{}, len(defaultValue))
	for _i := range defaultValue {
		_va[_i] = defaultValue[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetSlice")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...[]string) ([]string, error)); ok {
		return rf(key, defaultValue...)
	}
	if rf, ok := ret.Get(0).(func(biz.SettingKey, ...[]string) []string); ok {
		r0 = rf(key, defaultValue...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(biz.SettingKey, ...[]string) error); ok {
		r1 = rf(key, defaultValue...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SettingRepo_GetSlice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSlice'
type SettingRepo_GetSlice_Call struct {
	*mock.Call
}

// GetSlice is a helper method to define mock.On call
//   - key biz.SettingKey
//   - defaultValue ...[]string
func (_e *SettingRepo_Expecter) GetSlice(key interface{}, defaultValue ...interface{}) *SettingRepo_GetSlice_Call {
	return &SettingRepo_GetSlice_Call{Call: _e.mock.On("GetSlice",
		append([]interface{}{key}, defaultValue...)...)}
}

func (_c *SettingRepo_GetSlice_Call) Run(run func(key biz.SettingKey, defaultValue ...[]string)) *SettingRepo_GetSlice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([][]string, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.([]string)
			}
		}
		run(args[0].(biz.SettingKey), variadicArgs...)
	})
	return _c
}

func (_c *SettingRepo_GetSlice_Call) Return(_a0 []string, _a1 error) *SettingRepo_GetSlice_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SettingRepo_GetSlice_Call) RunAndReturn(run func(biz.SettingKey, ...[]string) ([]string, error)) *SettingRepo_GetSlice_Call {
	_c.Call.Return(run)
	return _c
}

// Set provides a mock function with given fields: key, value
func (_m *SettingRepo) Set(key biz.SettingKey, value string) error {
	ret := _m.Called(key, value)

	if len(ret) == 0 {
		panic("no return value specified for Set")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(biz.SettingKey, string) error); ok {
		r0 = rf(key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SettingRepo_Set_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Set'
type SettingRepo_Set_Call struct {
	*mock.Call
}

// Set is a helper method to define mock.On call
//   - key biz.SettingKey
//   - value string
func (_e *SettingRepo_Expecter) Set(key interface{}, value interface{}) *SettingRepo_Set_Call {
	return &SettingRepo_Set_Call{Call: _e.mock.On("Set", key, value)}
}

func (_c *SettingRepo_Set_Call) Run(run func(key biz.SettingKey, value string)) *SettingRepo_Set_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(biz.SettingKey), args[1].(string))
	})
	return _c
}

func (_c *SettingRepo_Set_Call) Return(_a0 error) *SettingRepo_Set_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SettingRepo_Set_Call) RunAndReturn(run func(biz.SettingKey, string) error) *SettingRepo_Set_Call {
	_c.Call.Return(run)
	return _c
}

// SetSlice provides a mock function with given fields: key, value
func (_m *SettingRepo) SetSlice(key biz.SettingKey, value []string) error {
	ret := _m.Called(key, value)

	if len(ret) == 0 {
		panic("no return value specified for SetSlice")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(biz.SettingKey, []string) error); ok {
		r0 = rf(key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SettingRepo_SetSlice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetSlice'
type SettingRepo_SetSlice_Call struct {
	*mock.Call
}

// SetSlice is a helper method to define mock.On call
//   - key biz.SettingKey
//   - value []string
func (_e *SettingRepo_Expecter) SetSlice(key interface{}, value interface{}) *SettingRepo_SetSlice_Call {
	return &SettingRepo_SetSlice_Call{Call: _e.mock.On("SetSlice", key, value)}
}

func (_c *SettingRepo_SetSlice_Call) Run(run func(key biz.SettingKey, value []string)) *SettingRepo_SetSlice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(biz.SettingKey), args[1].([]string))
	})
	return _c
}

func (_c *SettingRepo_SetSlice_Call) Return(_a0 error) *SettingRepo_SetSlice_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SettingRepo_SetSlice_Call) RunAndReturn(run func(biz.SettingKey, []string) error) *SettingRepo_SetSlice_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateCert provides a mock function with given fields: req
func (_m *SettingRepo) UpdateCert(req *request.SettingCert) error {
	ret := _m.Called(req)

	if len(ret) == 0 {
		panic("no return value specified for UpdateCert")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*request.SettingCert) error); ok {
		r0 = rf(req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SettingRepo_UpdateCert_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateCert'
type SettingRepo_UpdateCert_Call struct {
	*mock.Call
}

// UpdateCert is a helper method to define mock.On call
//   - req *request.SettingCert
func (_e *SettingRepo_Expecter) UpdateCert(req interface{}) *SettingRepo_UpdateCert_Call {
	return &SettingRepo_UpdateCert_Call{Call: _e.mock.On("UpdateCert", req)}
}

func (_c *SettingRepo_UpdateCert_Call) Run(run func(req *request.SettingCert)) *SettingRepo_UpdateCert_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*request.SettingCert))
	})
	return _c
}

func (_c *SettingRepo_UpdateCert_Call) Return(_a0 error) *SettingRepo_UpdateCert_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SettingRepo_UpdateCert_Call) RunAndReturn(run func(*request.SettingCert) error) *SettingRepo_UpdateCert_Call {
	_c.Call.Return(run)
	return _c
}

// UpdatePanel provides a mock function with given fields: req
func (_m *SettingRepo) UpdatePanel(req *request.SettingPanel) (bool, error) {
	ret := _m.Called(req)

	if len(ret) == 0 {
		panic("no return value specified for UpdatePanel")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(*request.SettingPanel) (bool, error)); ok {
		return rf(req)
	}
	if rf, ok := ret.Get(0).(func(*request.SettingPanel) bool); ok {
		r0 = rf(req)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(*request.SettingPanel) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SettingRepo_UpdatePanel_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdatePanel'
type SettingRepo_UpdatePanel_Call struct {
	*mock.Call
}

// UpdatePanel is a helper method to define mock.On call
//   - req *request.SettingPanel
func (_e *SettingRepo_Expecter) UpdatePanel(req interface{}) *SettingRepo_UpdatePanel_Call {
	return &SettingRepo_UpdatePanel_Call{Call: _e.mock.On("UpdatePanel", req)}
}

func (_c *SettingRepo_UpdatePanel_Call) Run(run func(req *request.SettingPanel)) *SettingRepo_UpdatePanel_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*request.SettingPanel))
	})
	return _c
}

func (_c *SettingRepo_UpdatePanel_Call) Return(_a0 bool, _a1 error) *SettingRepo_UpdatePanel_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SettingRepo_UpdatePanel_Call) RunAndReturn(run func(*request.SettingPanel) (bool, error)) *SettingRepo_UpdatePanel_Call {
	_c.Call.Return(run)
	return _c
}

// NewSettingRepo creates a new instance of SettingRepo. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSettingRepo(t interface {
	mock.TestingT
	Cleanup(func())
}) *SettingRepo {
	mock := &SettingRepo{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
