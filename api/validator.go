package api

import (
	"github.com/andyrestart9/bank/util"
	"github.com/go-playground/validator/v10"
)

// 解釋 fieldLevel.Field().Interface().(string)
// v := 123                 // 具體型別 int
// rv := reflect.ValueOf(v) // ① 取得 reflect.Value ，和 fieldLevel.Field() 一樣，回傳 reflect.Value
// iface := rv.Interface()  // ② 轉成 interface{} ，把 reflect.Value轉成 interface{} 之後才能用型別斷言、傳遞到普通函式，或以非反射方式自由操作
// x, ok := iface.(int)     // ③ 型別斷言
// Field() 會回傳 reflect.Value
// reflect.Value 只能用反射 API 操作
var validCurrency validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if currency, ok := fieldLevel.Field().Interface().(string); ok {
		return util.IsSupportedCurrency(currency)
	}
	return false
}
