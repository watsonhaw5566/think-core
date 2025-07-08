package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ThinkResponse struct {
	resp *http.Response
}

type ThinkHttpClient struct {
	client  http.Client
	headers map[string]string
}

func NewClient(minute ...int) *ThinkHttpClient {
	duration := time.Duration(60) * time.Second
	if len(minute) > 0 {
		duration = time.Duration(minute[0]) * time.Second
	}
	client := &ThinkHttpClient{
		client: http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   5,                // 每个主机的最大空闲连接数
				MaxConnsPerHost:       100,              // 每个主机的最大并发连接数
				IdleConnTimeout:       90 * time.Second, // 空闲连接在关闭之前的最长保持时间
				TLSHandshakeTimeout:   10 * time.Second, // TLS握手的超时时间
				ExpectContinueTimeout: 1 * time.Second,  // 服务器在发送100 Continue响应之前等待的时间
			},
			Timeout: duration,
		},
		headers: make(map[string]string),
	}
	client.headers["Content-Type"] = "application/json"
	return client
}

func (c *ThinkHttpClient) SetHeader(header map[string]string) *ThinkHttpClient {
	for key, value := range header {
		c.headers[key] = value
	}
	return c
}

func (c *ThinkHttpClient) GET(url string, params ...map[string]interface{}) (*ThinkResponse, error) {
	if len(params) > 0 {
		url = url + "?" + c.toValues(params[0])
	}
	return c.doRequest("GET", url, nil)
}

func (c *ThinkHttpClient) POST(url string, params map[string]interface{}) (*ThinkResponse, error) {
	if c.hasFile(params) {
		return c.doRequest("POST", url, params)
	}
	return c.doRequest("POST", url, params)
}

func (c *ThinkHttpClient) PUT(url string, params map[string]interface{}) (*ThinkResponse, error) {
	return c.doRequest("PUT", url, params)
}

func (c *ThinkHttpClient) DELETE(url string, params map[string]interface{}) (*ThinkResponse, error) {
	return c.doRequest("DELETE", url, params)
}

func (r *ThinkResponse) ReadAllString() string {
	body, _ := io.ReadAll(r.resp.Body)
	defer r.resp.Body.Close()
	return string(body)
}

func (r *ThinkResponse) ReadAll() []byte {
	body, _ := io.ReadAll(r.resp.Body)
	defer r.resp.Body.Close()
	return body
}

func (c *ThinkHttpClient) doRequest(method string, path string, params map[string]interface{}) (*ThinkResponse, error) {
	var reader io.Reader
	contentType := c.headers["Content-Type"]
	switch {
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		reader = strings.NewReader(c.toValues(params))
	case strings.Contains(contentType, "application/json"):
		jsonData, _ := json.Marshal(params)
		reader = bytes.NewReader(jsonData)
	case strings.Contains(contentType, "multipart/form-data"):
		var buf bytes.Buffer
		multiWriter := multipart.NewWriter(&buf)
		for key, value := range params {
			switch v := value.(type) {
			case string, int:
				if err := multiWriter.WriteField(key, fmt.Sprintf("%v", v)); err != nil {
					return nil, err
				}
			case *os.File:
				if err := c.writeFormFile(multiWriter, key, v); err != nil {
					return nil, err
				}
			}
		}
		reader = &buf
		contentType = multiWriter.FormDataContentType()
		multiWriter.Close()
	}
	req, err := http.NewRequest(method, path, reader)
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
		if strings.Contains(contentType, "multipart/form-data") {
			req.Header.Set("Content-Type", contentType)
		}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return &ThinkResponse{resp: resp}, nil
}

func (c *ThinkHttpClient) writeFormFile(writer *multipart.Writer, key string, file *os.File) error {
	part, err := writer.CreateFormFile(key, file.Name())
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(part, file)
	return err
}

func (c *ThinkHttpClient) toValues(args map[string]interface{}) string {
	params := url.Values{}
	for key, value := range args {
		params.Set(key, fmt.Sprintf("%v", value))
	}
	return params.Encode()
}

func (c *ThinkHttpClient) hasFile(params map[string]interface{}) bool {
	for _, v := range params {
		if _, ok := v.(*os.File); ok {
			return true
		}
	}
	return false
}
