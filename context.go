package tg

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/think-go/tg/tgcfg"
	"github.com/think-go/tg/tglog"
	"github.com/tidwall/gjson"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Context 上下文
type Context struct {
	Response http.ResponseWriter
	Request  *http.Request
	index    int
	handlers []HandlerFunc
	engine   *Engine
	cache    map[string]any
	mutex    sync.RWMutex
}

// errorCode 定义错误码
type errorCode struct {
	VALIDATE    int
	TokenExpire int
	EXCEPTION   int
	MySqlError  int
}

// ErrorCode 初始化错误码
var ErrorCode = &errorCode{
	VALIDATE:    10001, // 验证类错误
	TokenExpire: 10002, // Token过期
	EXCEPTION:   20001, // 服务或代码异常类错误
	MySqlError:  20002, // mysql错误
}

// result 统一返回结果
type result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// SuccessOption 自定义
type SuccessOption struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// FailOption 自定义
type FailOption struct {
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
func (ctx *Context) Success(data interface{}, option ...SuccessOption) {
	config := SuccessOption{
		Code:    http.StatusOK,
		Message: "ok",
	}
	if len(option) > 0 {
		if option[0].Code != 0 {
			config.Code = option[0].Code
		}
		if option[0].Message != "" {
			config.Message = option[0].Message
		}
	}
	ctx.JSON(http.StatusOK, result{
		Code:    config.Code,
		Message: config.Message,
		Data:    data,
	})
}

// Fail 异常输出信息
func (ctx *Context) Fail(message string, option ...FailOption) {
	config := FailOption{
		StatusCode: http.StatusUnauthorized,
		ErrorCode:  ErrorCode.VALIDATE,
	}
	if len(option) > 0 {
		if option[0].StatusCode != 0 {
			config.StatusCode = option[0].StatusCode
		}
		if option[0].ErrorCode != 0 {
			config.ErrorCode = option[0].ErrorCode
		}
	}
	ctx.JSON(config.StatusCode, SuccessOption{
		Code:    config.ErrorCode,
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

// Set 写入缓存信息
func (ctx *Context) Set(key string, value any) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	if ctx.cache == nil {
		ctx.cache = make(map[string]any)
	}
	ctx.cache[key] = value
	return
}

// Get 读取缓存信息
func (ctx *Context) Get(key string) (value any, ok bool) {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	value, ok = ctx.cache[key]
	return
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
			tglog.Log().Error("MultipartForm异常")
			return gjson.Result{}
		}
	}
	contentType := ctx.Request.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			tglog.Log().Error("Body解析失败")
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
func (ctx *Context) PostDefaultForm(key string, value any, defaultFormMaxMemory ...int64) gjson.Result {
	val := ctx.PostForm(key, defaultFormMaxMemory...)
	if val.String() == "" {
		return gjson.Get(fmt.Sprintf(`{"%s":%v}`, key, value), key)
	}
	return val
}

// FormFile 获取文件
func (ctx *Context) FormFile(key string, defaultFormMaxMemory ...int64) *multipart.FileHeader {
	maxMemory := int64(32) << 20
	if len(defaultFormMaxMemory) > 0 {
		maxMemory = defaultFormMaxMemory[0] << 20
	}
	if err := ctx.Request.ParseMultipartForm(maxMemory); err != nil {
		if !errors.Is(err, http.ErrNotMultipart) {
			tglog.Log().Error("FormFiles获取文件失败")
		}
	}
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

func bindParams(ctx *Context, req any, values url.Values) {
	if ctx.Request.MultipartForm != nil {
		for key, _ := range ctx.Request.MultipartForm.File {
			values.Set(key, "")
		}
	}
	reqVal := reflect.ValueOf(req).Elem()
	reqType := reqVal.Type()
	for i := 0; i < reqType.NumField(); i++ {
		name := reqType.Field(i).Tag.Get("p")
		for key, value := range values {
			if name == key {
				fieldName := reqType.Field(i).Name
				fieldVal := reqVal.FieldByName(fieldName)
				if fieldVal.CanSet() {
					switch fieldVal.Type() {
					case reflect.TypeOf(""):
						fieldVal.SetString(value[0])
					case reflect.TypeOf([]string{}):
						fieldVal.Set(reflect.ValueOf(value))
					case reflect.TypeOf(0):
						v, err := strconv.Atoi(value[0])
						if err != nil {
							tglog.Log().Error(err)
						}
						fieldVal.SetInt(int64(v))
					case reflect.TypeOf([]int{}):
						intSlice := make([]int, len(value))
						for i, s := range value {
							num, err := strconv.Atoi(s)
							if err != nil {
								tglog.Log().Error(err)
							}
							intSlice[i] = num
						}
						fieldVal.Set(reflect.ValueOf(intSlice))
					case reflect.TypeOf(new(multipart.FileHeader)):
						fieldVal.Set(reflect.ValueOf(ctx.FormFile(key)))
					case reflect.TypeOf([]*multipart.FileHeader{}):
						fieldVal.Set(reflect.ValueOf(ctx.FormFiles(key)))
					}
				}
			}
		}
	}
	CheckParams(req)
}

// BindStructValidate 结构体参数映射,具有参数验证功能
func (ctx *Context) BindStructValidate(req any, defaultFormMaxMemory ...int64) {
	maxMemory := int64(32) << 20
	if len(defaultFormMaxMemory) > 0 {
		maxMemory = defaultFormMaxMemory[0] << 20
	}

	if ctx.Request.Method == http.MethodGet {
		bindParams(ctx, req, ctx.Request.URL.Query())
		return
	}

	contentType := ctx.Request.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			panic(Exception{
				StateCode: http.StatusInternalServerError,
				ErrorCode: ErrorCode.EXCEPTION,
				Message:   "body解析失败",
				Error:     err,
			})
		}
		if err = json.Unmarshal(body, req); err != nil {
			panic(Exception{
				StateCode: http.StatusInternalServerError,
				ErrorCode: ErrorCode.EXCEPTION,
				Message:   "body映射失败",
				Error:     err,
			})
		}
		defer ctx.Request.Body.Close()
		CheckParams(req)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		bindParams(ctx, req, ctx.Request.PostForm)
	case strings.Contains(contentType, "multipart/form-data"):
		if err := ctx.Request.ParseMultipartForm(maxMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				panic(Exception{
					StateCode: http.StatusInternalServerError,
					ErrorCode: ErrorCode.EXCEPTION,
					Message:   "MultipartForm异常",
					Error:     err,
				})
			}
		}
		bindParams(ctx, req, ctx.Request.PostForm)
	default:
		CheckParams(req)
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
