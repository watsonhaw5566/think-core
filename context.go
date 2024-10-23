package tg

import (
	"encoding/json"
	"encoding/xml"
	"github.com/think-go/tg/tgcfg"
	"html/template"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Context 上下文
type Context struct {
	Response http.ResponseWriter
	Request  *http.Request
	index    int
	handlers []HandlerFunc
	latency  time.Duration
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
	ctx.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
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

// XML 输出XML
func (ctx *Context) XML(code int, data any) {
	// 设置响应头
	ctx.Response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	// 设置状态码
	ctx.Response.WriteHeader(code)
	// 将数据编码为XML
	xmlData, err := xml.Marshal(data)
	if err != nil {
		http.Error(ctx.Response, "服务异常解析失败", http.StatusInternalServerError)
		return
	}
	// 写入响应体
	ctx.Response.Write(xmlData)
}

// HTML 输出页面
func (ctx *Context) HTML(html string) {
	ctx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.Response.Write([]byte(html))
}

// View 输出页面模板
func (ctx *Context) View(name string, data any, expression ...string) {
	ctx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmp := template.New(name)
	tpl := "*.html"
	if len(expression) > 0 {
		tpl = expression[0]
	}
	tmp, err := tmp.ParseGlob(filepath.Join(tgcfg.Config.Server.TplPath, tpl))
	if err != nil {
		http.Error(ctx.Response, "服务异常解析失败", http.StatusInternalServerError)
		return
	}
	err = tmp.Execute(ctx.Response, data)
	if err != nil {
		http.Error(ctx.Response, "服务异常解析失败", http.StatusInternalServerError)
		return
	}
}

// Next 中间件向下执行
func (ctx *Context) Next() {
	ctx.index++
	if ctx.index < len(ctx.handlers) {
		start := time.Now()
		ctx.handlers[ctx.index](ctx)
		ctx.latency = time.Since(start)
	}
}

// ClientIP 获取IP
func (ctx *Context) ClientIP() string {
	ip := ctx.Request.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}
	ip = ctx.Request.Header.Get("X-Real-IP")
	if ip != "" {
		return strings.TrimSpace(ip)
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.Request.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

// Latency 用时
func (ctx *Context) Latency() time.Duration {
	return ctx.latency
}
