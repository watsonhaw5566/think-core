package tgtoken

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/think-go/tg"
	"net/http"
	"strings"
)

type JwtTokenOption struct {
	JwtKey    string
	Algorithm *jwt.SigningMethodHMAC
	Issuer    string
	Subject   string
	Audience  jwt.ClaimStrings
	ExpiresAt *jwt.NumericDate
	NotBefore *jwt.NumericDate
	IssuedAt  *jwt.NumericDate
	ID        string
}

type CustomClaims struct {
	Data any `json:"data"`
	jwt.RegisteredClaims
}

// CreateJwtToken 创建Token
func CreateJwtToken(data any, option JwtTokenOption) string {
	if option.Algorithm == nil {
		option.Algorithm = jwt.SigningMethodHS256
	}
	if option.JwtKey == "" {
		panic(tg.Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: tg.ErrorCode.EXCEPTION,
			Message:   "密钥不能为空",
		})
	}
	claims := jwt.NewWithClaims(option.Algorithm, CustomClaims{
		Data: data,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    option.Issuer,
			Subject:   option.Subject,
			Audience:  option.Audience,
			ExpiresAt: option.ExpiresAt,
			NotBefore: option.NotBefore,
			IssuedAt:  option.IssuedAt,
			ID:        option.ID,
		},
	})
	token, err := claims.SignedString([]byte(option.JwtKey))
	if err != nil {
		panic(tg.Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: tg.ErrorCode.EXCEPTION,
			Message:   "Token创建出错",
			Error:     err,
		})
	}
	return token
}

// GetAuthorization 从请求头中获取Token
func GetAuthorization(authorization string) string {
	str := strings.Split(authorization, " ")
	if len(str) != 2 || str[0] != "Bearer" {
		panic(tg.Exception{
			StateCode: http.StatusUnauthorized,
			ErrorCode: tg.ErrorCode.VALIDATE,
			Message:   "Token格式错误",
		})
	}
	return str[1]
}

// ParseToken 验证和解析Token
func ParseToken(tokenStr string, jwtKey string) string {
	claims := new(CustomClaims)
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})
	if err != nil {
		if err == jwt.ErrTokenExpired {
			panic(tg.Exception{
				StateCode: http.StatusUnauthorized,
				ErrorCode: tg.ErrorCode.TokenExpire,
				Message:   "Token已过期",
			})
		}
		panic(tg.Exception{
			StateCode: http.StatusUnauthorized,
			ErrorCode: tg.ErrorCode.VALIDATE,
			Message:   "Token解析出错",
			Error:     err,
		})
	}
	if !token.Valid {
		panic(tg.Exception{
			StateCode: http.StatusUnauthorized,
			ErrorCode: tg.ErrorCode.VALIDATE,
			Message:   "无效的Token",
		})
	}
	jsonBytes, err := json.Marshal(claims.Data)
	if err != nil {
		panic(tg.Exception{
			StateCode: http.StatusUnauthorized,
			ErrorCode: tg.ErrorCode.VALIDATE,
			Message:   "Token转json出错",
			Error:     err,
		})
	}
	return string(jsonBytes)
}
