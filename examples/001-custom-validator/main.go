package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin/binding"
)

// ---------- 預設驗證器 (示意 stub) ----------
type defaultValidator struct{}

func (d *defaultValidator) ValidateStruct(i interface{}) error { return nil }
func (d *defaultValidator) Engine() interface{}                { return "playground/validator" }

// ---------- 我們要替換成自己的驗證器 ----------
type myValidator struct{}

func (m *myValidator) ValidateStruct(i interface{}) error {
	// 只檢查 Owner != ""
	if v, ok := i.(Account); ok && v.Owner == "" {
		return fmt.Errorf("owner empty")
	}
	return nil
}
func (m *myValidator) Engine() interface{} { return "my custom engine" }

// ---------- 被驗證用的結構 ----------
type Account struct {
	ID    int64  `json:"id"`
	Owner string `json:"owner"`
}

func main() {
	// 預設情況：Gin 把 defaultValidator 放進 binding.Validator
	binding.Validator = &defaultValidator{}
	callFrameworkCode(Account{ID: 1, Owner: "Bob"})

	// 使用者自行替換
	binding.Validator = &myValidator{}
	callFrameworkCode(Account{ID: 2, Owner: ""}) // 將觸發自訂驗證錯誤
}

// ---------- 框架／共用程式碼 ----------
func callFrameworkCode(acc Account) {
	// 想用進階功能 → 執行期做型別斷言
	// 斷言成 string 是因為 func (m *myValidator) Engine() 返回的是 string
	if eng, ok := binding.Validator.Engine().(string); ok {
		log.Println("Underlying engine is:", eng)
	}

	// 編譯期：只知道 binding.Validator 是 StructValidator 介面
	if err := binding.Validator.ValidateStruct(acc); err != nil {
		log.Println("ValidateStruct error:", err)
		return
	}
}
