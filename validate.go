package think

import (
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTrans "github.com/go-playground/validator/v10/translations/zh"
	"github.com/think-go/tg/log"
	"net/http"
	"reflect"
)

var (
	uni      *ut.UniversalTranslator
	validate *validator.Validate
)

func init() {
	validate = validator.New()
	validate.SetTagName("v")
	uni = ut.New(zh.New(), en.New())
}

func CheckParams(req any) {
	trans, _ := uni.GetTranslator("zh")
	err := zhTrans.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		log.Log().Error(err)
	}
	if err = validate.Struct(req); err != nil {
		errs := err.(validator.ValidationErrors)
		for _, e := range errs {
			field, ok := reflect.TypeOf(req).Elem().FieldByName(e.Field())
			if ok {
				msg := field.Tag.Get("msg")
				if msg != "" {
					panic(Exception{
						StateCode: http.StatusUnauthorized,
						ErrorCode: ErrorCode.VALIDATE,
						Message:   msg,
					})
				}
			}
			panic(Exception{
				StateCode: http.StatusUnauthorized,
				ErrorCode: ErrorCode.VALIDATE,
				Message:   e.Translate(trans),
			})
		}
	}
}
