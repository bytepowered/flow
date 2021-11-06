package flow

import (
	"fmt"
	"os"
	"reflect"
)

const (
	assertPrefix = "FLOW:CRITICAL:ASSERT:"
)

var (
	_AssertEnabled = func() bool {
		// 允许通过环境变量禁用断言
		v, ok := os.LookupEnv("ASSERT_DISABLED")
		return !ok || "true" != v
	}()
)

func Assert(shouldTrue bool, errMessage string) {
	AssertTrue(shouldTrue, errMessage)
}

// AssertTrue 对输入bool值进行断言，期望为True；如果输入值为False，断言将触发panic，抛出错误消息。
func AssertTrue(shouldTrue bool, errMessage string) {
	if _AssertEnabled && !shouldTrue {
		panic(fmt.Errorf(assertPrefix+"%s", errMessage))
	}
}

// AssertFalse 对输入bool值进行断言，期望为False；如果输入值为True，断言将触发panic，抛出错误消息。
func AssertFalse(shouldFalse bool, errMessage string) {
	if _AssertEnabled && shouldFalse {
		panic(fmt.Errorf(assertPrefix+"%s", errMessage))
	}
}

// AssertNil 对输入值进行断言，期望为Nil(包含nil和值nil情况)；如果输入值为非Nil，断言将触发panic，抛出错误消息（消息模板）。
func AssertNil(v interface{}, message string, args ...interface{}) {
	if _AssertEnabled && NotNil(v) {
		panic(fmt.Errorf(assertPrefix+"%s", fmt.Sprintf(message, args...)))
	}
}

// AssertNotNil 对输入值进行断言，期望为非Nil(非nil并且值非nil的情况)；如果输入值为Nil，断言将触发panic，抛出错误消息（消息模板）。
func AssertNotNil(v interface{}, message string, args ...interface{}) {
	if _AssertEnabled && IsNil(v) {
		panic(fmt.Errorf(assertPrefix+"%s", fmt.Sprintf(message, args...)))
	}
}

// IsNil 判断输入值是否为Nil值（包括：nil、类型非Nil但值为Nil），用于检查类型值是否为Nil。
// 只针对引用类型判断有效，任何数值类型、结构体非指针类型等均为非Nil值。
func IsNil(i interface{}) bool {
	if nil == i {
		return true
	}
	value := reflect.ValueOf(i)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map,
		reflect.Interface, reflect.Slice,
		reflect.Ptr, reflect.UnsafePointer:
		return value.IsNil()
	}
	return false
}

// NotNil 判断输入值是否为非Nil值（包括：nil、类型非Nil但值为Nil），用于检查类型值是否为Nil。
// 只针对引用类型判断有效，任何数值类型、结构体非指针类型等均为非Nil值。
func NotNil(v interface{}) bool {
	return !IsNil(v)
}
