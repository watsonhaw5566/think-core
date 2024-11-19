package tghttp

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

type HttpClient struct {
	client  http.Client
	headers map[string]string
}

func NewClient(minute ...int) *HttpClient {
	duration := time.Duration(60) * time.Second
	if len(minute) > 0 {
		duration = time.Duration(minute[0]) * time.Second
	}
	client := &HttpClient{
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

func (c *HttpClient) SetHeader(header map[string]string) *HttpClient {
	for key, value := range header {
		c.headers[key] = value
	}
	return c
}

func (c *HttpClient) GET(url string, params ...map[string]interface{}) (io.ReadCloser, error) {
	if len(params) > 0 {
		url = url + "?" + c.toValues(params[0])
	}
	return c.doRequest("GET", url, nil)
}

func (c *HttpClient) POST(url string, params map[string]interface{}) (io.ReadCloser, error) {
	if c.hasFile(params) {
		return c.doRequest("POST", url, params)
	}
	return c.doRequest("POST", url, c.toValues(params))
}

func (c *HttpClient) PUT(url string, params map[string]interface{}) (io.ReadCloser, error) {
	return c.doRequest("PUT", url, c.toValues(params))
}

func (c *HttpClient) DELETE(url string, params map[string]interface{}) (io.ReadCloser, error) {
	return c.doRequest("DELETE", url, c.toValues(params))
}

func (c *HttpClient) doRequest(method string, path string, params interface{}) (io.ReadCloser, error) {
	var reader io.Reader
	switch c.headers["Content-Type"] {
	case "application/x-www-form-urlencoded":
		reader = strings.NewReader(params.(string))
	case "application/json":
		jsonData, _ := json.Marshal(params)
		reader = bytes.NewReader(jsonData)
	case "multipart/form-data":
		var buf bytes.Buffer
		multiWriter := multipart.NewWriter(&buf)
		formData := params.(map[string]interface{})
		for key, value := range formData {
			switch v := value.(type) {
			case string:
				if err := multiWriter.WriteField(key, v); err != nil {
					return nil, err
				}
			case *os.File:
				if err := c.writeFormFile(multiWriter, key, v); err != nil {
					return nil, err
				}
			}
		}
		if err := multiWriter.Close(); err != nil {
			return nil, err
		}
		reader = &buf
	}
	req, err := http.NewRequest(method, path, reader)
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (c *HttpClient) writeFormFile(writer *multipart.Writer, key string, file *os.File) error {
	part, err := writer.CreateFormFile(key, file.Name())
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(part, file)
	return err
}

func (c *HttpClient) toValues(args map[string]interface{}) string {
	params := url.Values{}
	for key, value := range args {
		params.Set(key, fmt.Sprintf("%v", value))
	}
	return params.Encode()
}

func (c *HttpClient) hasFile(params map[string]interface{}) bool {
	for _, v := range params {
		if _, ok := v.(*os.File); ok {
			return true
		}
	}
	return false
}
