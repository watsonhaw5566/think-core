package tgutl

import (
	"crypto/aes"
	"crypto/cipher"
	rand2 "crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/think-go/tg/tgcfg"
	"io"
	"math/rand"
	"strings"
	"time"
)

// HasSuffix 判断路由后缀是否在文件类型组里
func HasSuffix(url string) bool {
	staticPrefix := strings.Split(tgcfg.Config.Server.StaticSuffix, ",")
	for _, prefix := range staticPrefix {
		if strings.HasSuffix(url, "."+prefix) {
			return true
		}
	}
	return false
}

// GenerateSMSCode 生成短信验证码
func GenerateSMSCode(length int) int {
	rand.Seed(time.Now().UnixNano()) // 初始化随机数生成器
	code := 0
	multiplier := 1
	for i := 0; i < length; i++ {
		digit := rand.Intn(10) // 生成0到9的数字
		code += digit * multiplier
		multiplier *= 10
	}
	return code
}

// GetRandomNumber 生成m-n之间的随机整数
func GetRandomNumber(m int, n int) (res int) {
	rand.Seed(time.Now().UnixNano())
	res = rand.Intn(n) + m
	return
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) string {
	// 定义包含所有可能字符的切片
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// 设置随机数种子
	rand.Seed(time.Now().UnixNano())

	// 随机打乱切片
	chars := []byte(charset)
	for i := len(chars) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		chars[i], chars[j] = chars[j], chars[i]
	}

	// 从打乱后的切片中取出前 length 个字符作为随机字符串
	randomString := string(chars[:length])

	return randomString
}

// GetOutTradeNo 生成订单号
func GetOutTradeNo(businessPrefix string) string {
	// 获取当前日期时间，用于订单号的一部分
	now := time.Now().Format("20060102150405") // 格式化为年月日时分秒，例如：20230725141836

	// 生成随机数，可以使用更复杂的随机算法
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(1000) // 生成0-999的随机数

	// 组合订单号
	outTradeNo := fmt.Sprintf("%s%s%d", businessPrefix, now, randomNumber)
	return outTradeNo
}

// IndexOf 获取数组下标
func IndexOf(arr []string, value string) int {
	for i, v := range arr {
		if v == value {
			return i
		}
	}
	return -1
}

// Encrypt 字符串加密
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 使用CBC模式加密
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand2.Reader, iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))

	// 返回Base64编码的加密结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 字符串解密
func Decrypt(ciphertext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Base64解码
	encryptedData, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// 使用CBC模式解密
	iv := encryptedData[:aes.BlockSize]
	encryptedData = encryptedData[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(encryptedData, encryptedData)

	return string(encryptedData), nil
}

// StringInSlice 验证字符串数组中是否存在某字符串
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
