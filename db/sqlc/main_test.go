package db // 定義此檔案屬於 db 套件，用於存放資料庫相關邏輯

import (
	"database/sql" // Go 標準庫中的通用資料庫介面
	"log"          // 用於日誌輸出
	"os"           // 用於存取作業系統功能，例如退出程式
	"testing"      // Go 的測試框架

	_ "github.com/lib/pq" // 匿名匯入 PostgreSQL 驅動，藉由 init() 自動註冊到 database/sql
)

const (
	dbDriver = "postgres"                                                     // 資料庫驅動名稱，對應 github.com/lib/pq 所註冊的 driver
	dbSource = "postgresql://root:secret@localhost:5432/bank?sslmode=disable" // DSN (Data Source Name)：連線字串，包含使用者、密碼、主機、埠號、資料庫名稱及其他參數
	//   - "postgresql://"：指定使用 PostgreSQL 協定
	//   - "root:secret"：使用者帳號 root，密碼 secret
	//   - "@localhost:5432"：資料庫伺服器在本機的 5432 埠
	//   - "/bank"：要連接的資料庫名稱為 bank
	//   - "?sslmode=disable"：不使用 SSL（通常在開發或測試環境下）
)

var testQueries *Queries // 全域變數，儲存封裝過的 SQL 查詢物件，供各測試函式共用
var testDB *sql.DB // 全域變數，儲存資料庫連線物件，供各測試函式共用

// TestMain 是 Go testing 套件識別的特殊函式，會在執行此 package 底下所有測試之前先呼叫
func TestMain(m *testing.M) {
	var err error

	// 開啟資料庫連線（只是初始化 *sql.DB，不會立即與資料庫建立 TCP 連線），只會檢查 driver 和 DSN 是否有效，並返回一個 *sql.DB 物件，這個物件內部只是維護了一個「連線池」的抽象結構
	testDB, err = sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err) // 若 driver 或 DSN 有誤，立即記錄錯誤並結束程式
	}
	// 使用自訂函式 New(testDB) 來建立 *Queries 實例，將資料庫連線物件傳入
	// Queries 通常封裝所有 SQL 方法（如 CreateAccount、GetUser 等）
	testQueries = New(testDB)

	// 執行所有測試函式，並以它回傳的代碼作為程式退出碼
	os.Exit(m.Run())
}
