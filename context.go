package tg

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/think-go/tg/tgcfg"
	"github.com/think-go/tg/tglog"
	"github.com/tidwall/gjson"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
)

// Context 上下文
type Context struct {
	Response http.ResponseWriter
	Request  *http.Request
	index    int
	handlers []HandlerFunc
	engine   *Engine
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
	Data    interface{} `json:"data"`
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

// Exception 统一异常
type Exception struct {
	StateCode int    `json:"stateCode"`
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	Error     error  `json:"error"`
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
	ctx.JSON(http.StatusOK, result{
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
	ctx.JSON(statusCode, SuccessOptions{
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

// GetQuery 获取GET请求参数
func (ctx *Context) GetQuery(key string) string {
	return ctx.Request.URL.Query().Get(key)
}

// GetDefaultQuery 获取GET请求参数,如果没有内容赋默认值
func (ctx *Context) GetDefaultQuery(key string, value string) string {
	val := ctx.GetQuery(key)
	if val == "" {
		return value
	}
	return val
}

// PostForm 获取POST请求参数
func (ctx *Context) PostForm(key string, defaultFormMaxMemory ...int64) gjson.Result {
	maxMemory := int64(32) << 20
	if len(defaultFormMaxMemory) > 0 {
		maxMemory = defaultFormMaxMemory[0] << 20
	}
	if err := ctx.Request.ParseMultipartForm(maxMemory); err != nil {
		if !errors.Is(err, http.ErrNotMultipart) {
			tglog.Log().Error("POST获取参数失败")
			return gjson.Result{}
		}
	}
	contentType := ctx.Request.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			tglog.Log().Error("POST获取参数失败")
			return gjson.Result{}
		}
		defer ctx.Request.Body.Close()
		return gjson.Get(string(body), key)
	}
	value := ctx.Request.PostForm.Get(key)
	if value == "" {
		return gjson.Result{}
	}
	return gjson.Get(fmt.Sprintf(`{"%s":"%s"}`, key, value), key)
}

// PostDefaultForm 获取POST请求参数,如果没有内容赋默认值
func (ctx *Context) PostDefaultForm(key string, value any) gjson.Result {
	val := ctx.PostForm(key)
	if val.String() == "" {
		return gjson.Get(fmt.Sprintf(`{"%s":%v}`, key, value), key)
	}
	return val
}

// FormFile 获取文件
func (ctx *Context) FormFile(key string) *multipart.FileHeader {
	file, header, err := ctx.Request.FormFile(key)
	if err != nil {
		tglog.Log().Error(err)
		return nil
	}
	defer file.Close()
	return header
}

// FormFiles 获取多个文件
func (ctx *Context) FormFiles(key string, defaultFormMaxMemory ...int64) []*multipart.FileHeader {
	maxMemory := int64(32) << 20
	if len(defaultFormMaxMemory) > 0 {
		maxMemory = defaultFormMaxMemory[0] << 20
	}
	if err := ctx.Request.ParseMultipartForm(maxMemory); err != nil {
		if !errors.Is(err, http.ErrNotMultipart) {
			tglog.Log().Error("FormFiles获取多文件失败")
			return []*multipart.FileHeader{}
		}
	}
	files := ctx.Request.MultipartForm.File[key]
	if files == nil {
		return []*multipart.FileHeader{}
	}
	return files
}

// BindStructValidate 结构体参数映射,具有参数验证功能
func (ctx *Context) BindStructValidate(req any, defaultFormMaxMemory ...int64) {
	contentType := ctx.Request.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body := ctx.Request.Body
		if body == nil {
			panic(&Exception{
				StateCode: http.StatusBadRequest,
				ErrorCode: ErrorCode.VALIDATE,
				Message:   "body格式错误",
			})
		}
		decoder := json.NewDecoder(body)
		defer ctx.Request.Body.Close()
		err := decoder.Decode(req)
		if err != nil {
			panic(&Exception{
				StateCode: http.StatusBadRequest,
				ErrorCode: ErrorCode.VALIDATE,
				Message:   "结构体映射出错",
				Error:     err,
			})
		}
		// 验证
		validate(req)
	}
	// GET或POST的FormData非json情况
	switch ctx.Request.Method {
	case http.MethodGet:
		params := make(map[string]interface{})
		for key, values := range ctx.Request.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
		err := mapstructure.Decode(params, req)
		if err != nil {
			panic(&Exception{
				StateCode: http.StatusBadRequest,
				ErrorCode: ErrorCode.VALIDATE,
				Message:   "结构体映射出错",
				Error:     err,
			})
		}
		// 验证
		validate(req)
	case http.MethodPost, http.MethodPut, http.MethodDelete:
		rv := reflect.ValueOf(req)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			panic(&Exception{
				StateCode: http.StatusBadRequest,
				ErrorCode: ErrorCode.EXCEPTION,
				Message:   "传入必须是指针",
			})
		}
		// 获取指针指向的元素
		elem := rv.Elem()
		// 获取结构体定义
		st := reflect.TypeOf(req).Elem()
		for i := 0; i < elem.NumField(); i++ {
			key := st.Field(i).Tag.Get("p")
			switch st.Field(i).Type {
			case reflect.TypeOf(new(multipart.FileHeader)):
				elem.Field(i).Set(reflect.ValueOf(ctx.FormFile(key)))
			case reflect.TypeOf([]*multipart.FileHeader{}):
				elem.Field(i).Set(reflect.ValueOf(ctx.FormFiles(key, defaultFormMaxMemory...)))
			case reflect.TypeOf(0):
				elem.Field(i).SetInt(ctx.PostForm(key, defaultFormMaxMemory...).Int())
			case reflect.TypeOf(""):
				elem.Field(i).SetString(ctx.PostForm(key, defaultFormMaxMemory...).String())
			}
		}
		// 验证
		validate(req)
	}
}

// Stream 流式数据转发
func (ctx *Context) Stream(data io.Reader) {
	ctx.Response.Header().Set("Content-Type", "text/event-stream")
	ctx.Response.Header().Set("Cache-Control", "no-cache")
	ctx.Response.Header().Set("Connection", "keep-alive")
	if _, err := io.Copy(ctx.Response, data); err != nil {
		http.Error(ctx.Response, "服务异常解析失败", http.StatusInternalServerError)
		return
	}
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

// Download 文件下载
func (ctx *Context) Download(fileName string) {
	filePath := filepath.Join(tgcfg.Config.Server.StaticPath, fileName)
	ctx.Response.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	http.ServeFile(ctx.Response, ctx.Request, filePath)
}

// Redirect 重定向
func (ctx *Context) Redirect(url string) {
	http.Redirect(ctx.Response, ctx.Request, url, http.StatusFound)
}

// Next 中间件向下执行
func (ctx *Context) Next() {
	ctx.index++
	if ctx.index < len(ctx.handlers) {
		ctx.handlers[ctx.index](ctx)
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
