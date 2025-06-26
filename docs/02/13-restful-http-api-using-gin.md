# Implement RESTful HTTP API using Gin

## Install

<https://github.com/gin-gonic/gin?tab=readme-ov-file#getting-started>

`go get -u github.com/gin-gonic/gin`

## Model binding and validation

<https://gin-gonic.com/en/docs/examples/binding-and-validation/>

- Type - Should bind
  - Methods - ShouldBind, ShouldBindJSON, ShouldBindXML, ShouldBindQuery, ShouldBindYAML
  - Behavior - These methods use ShouldBindWith under the hood. If there is a binding error, the error is returned and it is the developer’s responsibility to handle the request and error appropriately.

You can also specify that specific fields are required. If a field is decorated with binding:"required" and has a empty value when binding, an error will be returned.

<https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-One_Of>

For strings, ints, and uints, oneof will ensure that the value is one of the values in the parameter.

Example:

currency 欄位必填，必須是 USD 或 EUR

```go
type createAccountRequest struct {
    Currency string `json:"currency" binding:"required,oneof=USD EUR"`
}

func (server *Server) createAccount(ctx *gin.Context) {
    var req createAccountRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }
}
```

## Bind Uri

<https://gin-gonic.com/en/docs/examples/bind-uri/>

<https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Minimum>

For numbers, min will ensure that the value is greater or equal to the parameter given. For strings, it checks that the string length is at least that number of characters. For slices, arrays, and maps, validates the number of items.

Example #1

`Usage: min=10`

Example #2 (time.Duration)

For time.Duration, min will ensure that the value is greater than or equal to the duration given in the parameter.

`Usage: min=1h30m`

Example:

ID 欄位必填，最小值為1

```go
type getAccountRequest struct {
    ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getAccount(ctx *gin.Context) {
    var req getAccountRequest
    if err := ctx.ShouldBindUri(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }
    // ...
}
```

## Only bind query string

<https://gin-gonic.com/en/docs/examples/only-bind-query-string/>

Example

```go
type Person struct {
  Name    string `form:"name"`
  Address string `form:"address"`
}

func startPage(c *gin.Context) {
  var person Person
  if c.ShouldBindQuery(&person) == nil {
    log.Println(person.Name)
    log.Println(person.Address)
  }
  c.String(200, "Success")
}
```

<https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Maximum>

For numbers, max will ensure that the value is less than or equal to the parameter given. For strings, it checks that the string length is at most that number of characters. For slices, arrays, and maps, validates the number of items.

Example #1

`Usage: max=10`

Example #2 (time.Duration)

For time.Duration, max will ensure that the value is less than or equal to the duration given in the parameter.

`Usage: max=1h30m`

## URI 和 Body 應該要分開綁定

RequestURI RequestBody 要分開綁定，因為 ShouldBindUri 会把“整份结构体”一次性做校验。当它发现结构体里还有 Balance 字段并带着 binding:"required"，就会先检查它——而这时 Balance 还没填，等于 0，于是 required 规则立刻失败

所以不能這樣做

```go
type updateAccountRequest struct {
    ID      int64 `uri:"id" binding:"required,min=1"`
    Balance int64 `json:"balance" binding:"required,min=0"`
}
```

而是要

```go
type updateAccountRequestURI struct {
     ID int64 `uri:"id" binding:"required,min=1"`
}

type updateAccountRequestBody struct {
     Balance int64 `json:"balance" binding:"required,min=0"`
}


func (server *Server) updateAccount(ctx *gin.Context) {
    var req updateAccountRequestURI
    if err := ctx.ShouldBindUri(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    var reqBody updateAccountRequestBody
    if err := ctx.ShouldBindJSON(&reqBody); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    // ...
}

```

## 如何讓 list account return 空 slice

sqlc 預設 ListAccounts 查到 0 筆資料返回 nil

如果要讓 sqlc 生成的 ListAccounts 在查到 0 筆資料時返回空 slice 要改 sqlc.yaml 設定

```yaml
emit_empty_slices: true # 如果为 true，则 :many 查询返回的切片将为空而不是 nil
```
