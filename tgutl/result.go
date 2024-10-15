package tgutl

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// errorCode 定义错误码
type errorCode struct {
	VALIDATE  int
	EXCEPTION int
}

// ErrorCode 初始化错误码
var ErrorCode = &errorCode{
	VALIDATE:  10001, // 验证类错误
	EXCEPTION: 20001, // 服务或代码异常类错误
}

// Result 统一返回结果
type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SuccessOptions 自定义
type SuccessOptions struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// FailOptions 自定义
type FailOptions struct {
	StatusCode int `json:"statusCode"`
	ErrorCode  int `json:"errorCode"`
}

// Success 成功输出信息
func Success(ctx *gin.Context, data interface{}, options ...SuccessOptions) {
	var opt SuccessOptions
	if len(options) > 0 {
		opt = options[0]
	}
	code := opt.Code
	if code == 0 {
		code = http.StatusOK
	}
	message := opt.Message
	if message == "" {
		message = "ok"
	}
	ctx.JSON(http.StatusOK, &Result{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Fail 异常输出信息
func Fail(ctx *gin.Context, message string, options ...FailOptions) {
	var opt FailOptions
	if len(options) > 0 {
		opt = options[0]
	}
	statusCode := opt.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	errCode := opt.ErrorCode
	if errCode == 0 {
		errCode = ErrorCode.VALIDATE
	}
	ctx.JSON(statusCode, &SuccessOptions{
		Code:    errCode,
		Message: message,
	})
}
