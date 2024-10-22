package tg

import (
	"encoding/json"
	"net/http"
)

// Context 上下文
type Context struct {
	Response http.ResponseWriter
	Request  *http.Request
	index    int
	handlers []HandlerFunc
}

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

// result 统一返回结果
type result struct {
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
func (ctx *Context) Success(data interface{}, options ...SuccessOptions) {
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
	ctx.JSON(http.StatusOK, &result{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Fail 异常输出信息
func (ctx *Context) Fail(message string, options ...FailOptions) {
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

// JSON 输出JSON
func (ctx *Context) JSON(code int, data any) {
	// 设置响应头
	ctx.Response.Header().Set("Content-Type", "application/json")
	// 设置状态码
	ctx.Response.WriteHeader(code)
	// 将数据编码为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(ctx.Response, "服务异常解析失败", http.StatusInternalServerError)
		return
	}
	// 写入响应体
	ctx.Response.Write(jsonData)
}

// Next 中间件向下执行
func (ctx *Context) Next() {
	ctx.index++
	if ctx.index < len(ctx.handlers) {
		ctx.handlers[ctx.index](ctx)
	}
}
