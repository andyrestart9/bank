# Write tests for database

## Install database driver

<https://pkg.go.dev/database/sql#Open>

Most users will open a database via a driver-specific connection helper function that returns a *DB. No database drivers are included in the Go standard library. See <https://golang.org/s/sqldrivers> for a list of third-party drivers.

Go 標準庫 "database/sql" 中不包含任何資料庫驅動程式

所以我們要自己安裝資料庫驅動程式 <https://github.com/lib/pq?tab=readme-ov-file#install>

`go get github.com/lib/pq`

### 如果沒有安裝資料庫驅動程式會怎麼樣？

`go test ./db/sqlc -v`

執行測試會出現 cannot connect to db:sql: unknown driver "postgres"

```sh
andylin@Andys-MacBook-Pro bank % go test ./db/sqlc -v
2025/06/01 22:14:37 cannot connect to db:sql: unknown driver "postgres" (forgotten import?)
FAIL    github.com/andyrestart9/bank/db/sqlc    0.155s
FAIL
```

## Install testify

<https://github.com/stretchr/testify?tab=readme-ov-file#installation>

比使用原本 if else statements 去寫測試更為簡潔

`go get github.com/stretchr/testify`

## 撰寫生成隨機字串和數字的函數用於產生測試資料

```go
package util

import (
 "math/rand"   // Go 標準庫的隨機數產生套件，提供 Intn、Int63n 等方法
 "strings"     // 用於操作字串，這裡主要使用 strings.Builder 來高效拼接字串
 "time"        // 用於取得系統時間，作為隨機數種子的來源
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" 
// 定義了一個包含 52 個英文大小寫字母的常量字串，用於生成隨機英文字母字串

// 建立一個本地的 RNG(Random Number Generator)，避免使用全局的 rand.Seed
// rand.NewSource(time.Now().UnixNano()) 會返回一個基於當前系統時間（納秒級）做種子的隨機源
// rand.New(...) 會使用這個隨機源建立一個 *rand.Rand 實例，後續所有隨機調用都透過 rng 完成
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// RandomInt generates a random integer between min and max
// 這個函式想要從閉區間 [min, max] 中得到一個隨機 int64 值
// rng.Int63n(n) 會產生一個 [0, n) 范圍內的隨機 int64，最高位永遠是 0，保證非負
// 為了讓結果能落在 [min, max]，需要讓 rng.Int63n 的參數是 (max - min + 1)
// 因為 rng.Int63n(max-min+1) 會回傳 0 ~ (max-min) 之間的整數，加上 min 後就變成 min ~ max
func RandomInt(min, max int64) int64 {
 return min + rng.Int63n(max-min+1)
}

// RandomString generates a random string of length n
// 如果 zh 為 true，就生成 n 個隨機中文漢字；否則生成 n 個隨機英文大小寫字母
func RandomString(n int, zh bool) string {
 var sb strings.Builder // strings.Builder 用於高效地逐個追加字元/字節

 if zh {
  // 用於生成「常用漢字」的 Unicode 區間是 0x4E00 到 0x9FA5
  // rng.Intn(0x9FA5-0x4E00+1) 會產生一個 [0, 0x9FA5-0x4E00] 的隨機 int
  // 加上 0x4E00 後就得到一個 [0x4E00, 0x9FA5] 區間內的隨機碼點
  for i := 0; i < n; i++ {
   // rune(runeValue) 轉成 int32 的 Unicode 碼點
   // WriteRune(rune) 會把 rune 自動編碼成 UTF-8（1~4 個位元組）後寫入 Builder
   sb.WriteRune(rune(0x4E00 + rng.Intn(0x9FA5-0x4E00+1)))
  }
 } else {
  // len(alphabet) = 52，代表我們總共有 52 個可選字母
  // rng.Intn(k) 會回傳一個 [0, k) 的隨機 int
  // WriteByte(byte) 直接把一個字節 (ASCII 范圍內) 寫入 Builder
  k := len(alphabet)
  for i := 0; i < n; i++ {
   sb.WriteByte(alphabet[rng.Intn(k)])
  }
 }

 // 將 Builder 裡累積的位元組轉成字串並返回
 return sb.String()
}
```

## 撰寫單元測試

參考： db/sqlc/account_test.go, db/sqlc/entry_test.go, db/sqlc/transfer_test.go

把新增單獨拆開來寫，因為在驗證查詢、更新、刪除都會需要先新增之後再操作

## Run test & package tests

寫好單元測試後在 func TestXxx 上面有一個 run test 可以按，按下去就會跑測試了

在 package xxx 上面有一個 run package tests ，按下去就會把這個 package 有的測試跑過一次，之後就會發現在被測試的檔案裡面有些代碼會有高亮顯示，這是 VS Code 的 Go 插件在做「测试覆盖率」高亮，绿底（covered）表示这行代码在你刚跑的测试里被执行到了，红底（uncovered）表示测试没跑到，編輯器的 output 也會顯示測試覆蓋率
