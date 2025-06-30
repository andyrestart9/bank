# 光速未來 - Go筆試考題_06/27林循安_11:00

## 一、請填入時間複雜度

|  | Access | Search | Insertion | Deletion |
| :---- | :---- | :---- | :---- | :---- |
| Dynamic Array | O(1) | O(n) | O(n) | O(n) |
| Linked List | O(n) | O(n) | O(1) | O(1) |
| Hash Table | O(1) | O(1) | O(1) | O(1) |

## 二、請寫出輸出結果，並給出原因

### 1

```go
func arr_set(arr []int) {
    arr[0] = 123
}
func arr_append(arr []int) {
    arr = append(arr, 777)
    fmt.Println(arr)
}
func main() {
    arr := []int{1, 2, 3}
    arr_set(arr)
    arr_append(arr)
    fmt.Println(arr)
}
```

[123 2 3 777]
[123 2 3]

arr_set 把 []int{1, 2, 3} 0 索引改成 123 ， append 會把底層陣列複製過去再加上 777

main 的變數 arr 沒有被更改，仍舊指向原本的陣列 [123 2 3]

### 2

```go
type Cat struct {
 name string
}

func main() {
 var i interface{}
 fmt.Println(i == nil)

 var p *Cat = nil
 i = p
 fmt.Println(i == nil)
}
```

true
false

型別和值都為 nil 時才會與 nil 相等

i = p 後，型別是 *Cat ，所以 i 不等於 nil

### 3

```go
func main() {
    runtime.GOMAXPROCS(1)
    wg := sync.WaitGroup{}
    count := 10000
    wg.Add(count)
    ans := 0
    for i := 0; i < count; i++ {
        go func() {
            ans++
            wg.Done()
        }()
    }
    wg.Wait()
    fmt.Println(ans)
}
```

### 4

```go
func main() {
 ch := make(chan int, 1)
 ch <- 1
 tick := time.NewTicker(100 * time.Millisecond)
 time.Sleep(1 * time.Second)
 select {
 case <-tick.C:
  fmt.Println("a")
 case <-ch:
  fmt.Println("b")
 default:
  fmt.Println("c")
 }
}

```

### 5

```go
type obj struct {
    param int
}
func main() {
    m := make(map[int]*obj)
    m[1] = &obj{param: 1}
    m[2] = &obj{param: 2}
    m[3] = &obj{param: 3}

    for _, v := range m {
        if v.param == 2 {
            v = &obj{param: 999}
        }
        v.param += 70
    }
    fmt.Println(m[1].param)
    fmt.Println(m[2].param)
    fmt.Println(m[3].param) }

```

### 6

```go
func foo() (n int) {
 defer func() {
  n += 5
  fmt.Println("a:", n)
 }()

 defer func() {
  n += 3
  fmt.Println("b:", n)
 }()
 n = 10
 return n
}

func main() {
 fmt.Println("c:", foo())
}

```

## 三、Go 基礎問題

1. slice 與 append()： 切片底層結構、容量擴充策略與內存複製成本。  
2. map 併發安全性： 原生 map 是否讀寫安全？sync.Map 與鎖保護的比較。  
3. 垃圾回收 (GC)： 三色標記清除、Stop‑the‑World 影響與 GOGC 調優。  
4. Goroutine 與執行緒的差異？GMP模型是如何分配調度？  
5. 請舉例 Channel 的用途, 緩衝與非緩衝 channel 的行為差異？  
6. 競態偵測器： go test \-race 結果輸出格式與常見修復步驟。  
7. 效能剖析 (pprof)： 如何定位記憶體洩漏或 CPU 熱點。

## 四、資料庫與系統架構考題
>
> 自由選擇自己熟悉的領域回答

### 1. 交易（Transaction）與 ACID 性質

請闡述資料庫「交易」的概念，以及其四大核心特性──原子性（Atomicity）、一致性（Consistency）、隔離性（Isolation）與持久性（Durability）。  
並請舉出一個實務場景（例：銀行轉帳、電商下單扣庫存等），說明為何必須以交易確保資料一致性，以及您將如何在該場景中實作交易控制。

### 2. SQL 查詢效能優化

請條列您進行 SQL 查詢效能調校的完整流程與策略，例如：

1. 確認執行計畫（Explain/Analyze）  
2. 指標收集（I/O、CPU、回傳列數）  
3. 查詢重寫（子查詢改 JOIN、避免 SELECT \*）  
4. 資料分區、分表或物化檢視

並說明「在何種情況下」您會為欄位建立索引，以及選擇哪種類型的索引（B-Tree、Hash、GIN/GiST 等）最為適當，並闡述原因。

### 3. Redis 快取寫入策略與常見風險

以 Redis 為例，比較 Write-Through、Write-Around、Write-Back 三種寫入策略的運作機制、優缺點及適用場景。  
同時請定義並說明「快取雪崩（Cache Avalanche）」與「快取穿透（Cache Penetration）」這兩種風險，及其可能的防範做法。

### 4. Message Queue與訂閱／發布（Publish/Subscribe）模型

簡述在 Pub/Sub 架構下，消息佇列（Message Queue） 的核心運作流程（消息投遞、消費者拉取／推送、確認機制等），並舉出一個實際應用情境（例如：訂單事件驅動、即時通知推播、日誌收集管線等），說明使用消息佇列可帶來的具體效益。

### 5. 多人連線遊戲的系統設計

請與考官討論並提出一套 多人連線德州撲克（Texas Hold’em）遊戲後端系統的高階設計構想，內容可包含（但不限於）：

1. 整體架構（Service 切分、通訊協定、負載平衡）  
2. 遊戲狀態同步與一致性維護（房間管理、牌局流程）  
3. 資料儲存策略（即時資料 vs. 歷史資料）  
4. 擴充性與高可用性的考量

請透過圖示或文字條列方式，闡述設計決策的理由與潛在權衡。
