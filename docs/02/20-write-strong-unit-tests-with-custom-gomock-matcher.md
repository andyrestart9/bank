# Write strong unit tests with custom gomock matcher

## 了解 gomock.Eq

```go
CreateUser(gomock.Any(), gomock.Eq(arg)).
```

對上面 `gomock.Eq(arg)` command + 左鍵點擊 `Eq` Go to Definition

會跳轉到下面 package gomock 看到，接著對 Matcher 和 eqMatcher 右鍵 Go to Definition 看他們的定義

```go
func Eq(x interface{}) Matcher { return eqMatcher{x} }
```

Matcher：

```go
type Matcher interface {
    // Matches returns whether x is a match.
    Matches(x interface{}) bool

    // String describes what the matcher matches.
    String() string
}
```

eqMatcher：

```go
type eqMatcher struct {
    x interface{}
}
```

要做一個 custom matcher 要實作 Matcher interface ，所以接下來要看 eqMatcher 怎麼實作 Matcher interface

對上面 `eqMatcher` 右鍵  Go to References

會看到下面 eqMatcher struct 有兩個方法 Matches 和 String ，也就是 Matcher interface 定義的那兩個方法

```go
func (e eqMatcher) Matches(x interface{}) bool {
    // In case, some value is nil
    if e.x == nil || x == nil {
        return reflect.DeepEqual(e.x, x)
    }

    // Check if types assignable and convert them to common type
    x1Val := reflect.ValueOf(e.x)
    x2Val := reflect.ValueOf(x)

    if x1Val.Type().AssignableTo(x2Val.Type()) {
        x1ValConverted := x1Val.Convert(x2Val.Type())
        return reflect.DeepEqual(x1ValConverted.Interface(), x2Val.Interface())
    }

    return false
}

func (e eqMatcher) String() string {
    return fmt.Sprintf("is equal to %v (%T)", e.x, e.x)
}
```

- e.x：
  - 來源：你在 EXPECT() 時呼叫 gomock.Eq(arg)；
  - arg 就被存進 eqMatcher 的欄位 x，成為「預期值 (expected)」。
- x（Matches 的參數）
  - 來源：執行期 handler 執行

    ```go
    server.store.CreateUser(ctx, argFromHandler)
    ```

  - gomock 轉呼叫 eqMatcher.Matches(argFromHandler)，所以這裡的 x 就是 真正傳進 mock 的那個參數（got）。

最終 Matches 內部會比較

reflect.DeepEqual(e.x /\*expected \*/, x /\* got\*/)

來決定這次呼叫是否符合你的期待設定。

## 實作 gomock Matcher

```go
// eqCreateUserParamsMatcher 是一個自訂的 gomock Matcher，
// 專門用來比對 CreateUser() 被呼叫時傳入的 db.CreateUserParams，
// 並確保：
//  1. 密碼確實被正確雜湊 (利用 util.CheckPassword 驗證)
//  2. 除了雜湊值之外，其餘欄位與預期完全一致
//
// 透過這種方式，我們可以避免直接比較雜湊後不一致的密碼字串。
type eqCreateUserParamsMatcher struct {
  // arg 保存「期望」傳進資料庫層的參數 (HashedPassword 先留空，稍後再補)
  arg db.CreateUserParams
  // password 是原始明文密碼，用來驗證 HashedPassword 是否正確
  password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
  // 1) 檢查傳進來的 x 是否為 db.CreateUserParams 型別
  arg, ok := x.(db.CreateUserParams)
  if !ok {
    return false
  }

  // 2) 用原始密碼驗證雜湊值是否正確
  err := util.CheckPassword(e.password, arg.HashedPassword)
  if err != nil {
    return false
  }

  // 3) 將實際產生的雜湊值寫回預期值，
  //    這樣後續 reflect.DeepEqual 才不會因 HashedPassword 不同而失敗
  e.arg.HashedPassword = arg.HashedPassword

  // 4) 用 reflect.DeepEqual 比對其餘所有欄位
  return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
  // 在比對失敗時，Gomock 會呼叫 String() 取得錯誤訊息，
  // 這裡回傳人類可讀的描述方便除錯
  return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
  // 工廠函式：建立並回傳一個已填好資料的 matcher 實例
  return eqCreateUserParamsMatcher{arg, password}
}
```
