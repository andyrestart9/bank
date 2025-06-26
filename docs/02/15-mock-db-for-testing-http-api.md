# Mock DB for testing HTTP API in Go and achieve 100% coverage

## Install the mockgen tool

<https://github.com/golang/mock?tab=readme-ov-file#installation>

`go install github.com/golang/mock/mockgen@v1.6.0` 下載 mockgen

`which mockgen` 確認 mockgen 指令能不能用

如果出現 mockgen not found 表示 $PATH 內沒有指向 mockgen 的二進制可執行檔所在的目錄， 用 `go env GOPATH` 或 `go env GOBIN` 看看 mockgen 被安裝到哪個目錄下，再把這個資目錄路徑加進 $PATH

## 讓 sqlc 生成 Querier interface

讓 sqlc 生成 Querier interface 我們可以在 Store interface 嵌入 Querier interface ，不用 Queries struct 的方法一個一個寫回去 Store interface

這也是 interface 的好處，只約定 struct 要實現哪些方法，但是 struct 可以是真的 DB 或是 mock DB

sqlc.yaml

```yaml
emit_interface: true # 生成 Querier 接口
```

db/sqlc/store.go

```go
type Store interface {
    Querier
}
```

## 如何使用 mockgen

顯示操作手冊 `mockgen -help`

generates mock interfaces:

`mockgen -package 指定package名稱 -destination 目標位址 module名稱/檔案路徑 interface名稱`

example: `mockgen -package mockdb -destination db/mock/store.go github.com/andyrestart9/bank/db/sqlc Store`

## 設定 gin 為 test mode

api/main_test.go

```go
func TestMain(m *testing.M) {
    gin.SetMode(gin.TestMode)
    os.Exit(m.Run())
}
```
