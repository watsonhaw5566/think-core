package tg

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
)

// validate 验证器
func validate(req any) {
	rv := reflect.ValueOf(req)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic(&Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: ErrorCode.EXCEPTION,
			Message:   "传入必须是指针",
		})
	}
	// 获取指针指向的元素
	elem := rv.Elem()
	// 获取结构体定义
	st := reflect.TypeOf(req).Elem()
	for i := 0; i < elem.NumField(); i++ {
		value := elem.Field(i)
		rule := st.Field(i).Tag.Get("v")
		name := st.Field(i).Tag.Get("p")
		if rule != "" {
			parts := strings.Split(rule, "#")
			switch parts[0] {
			case "required":
				msg := ""
				if len(parts) > 1 {
					msg = parts[1]
				}
				checkRequired(value, name, msg)
			default:
				panic(&Exception{
					StateCode: http.StatusInternalServerError,
					ErrorCode: ErrorCode.EXCEPTION,
					Message:   "未指定验证规则",
				})
			}
		}
	}
}

// checkRequired 验证必填
func checkRequired(value reflect.Value, name string, msg string) {
	if reflect.TypeOf(new(multipart.FileHeader)) == value.Type() {
		file := value.Interface().(*multipart.FileHeader)
		if file != nil {
			return
		}
	} else if reflect.TypeOf([]*multipart.FileHeader{}) == value.Type() {
		file := value.Interface().([]*multipart.FileHeader)
		if len(file) > 0 {
			return
		}
	} else if value.Kind() == reflect.String {
		if value.String() != "" {
			return
		}
	} else if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		if value.Len() <= 0 {
			return
		}
	}
	if msg == "" {
		msg = fmt.Sprintf("%s不能为空", name)
	}
	panic(&Exception{
		StateCode: http.StatusBadRequest,
		ErrorCode: ErrorCode.VALIDATE,
		Message:   msg,
	})
}
