package db

import (
	"context"      // 用於傳遞上下文，例如超時、取消等資訊
	"database/sql" // 提供 SQL 標準介面，這裡用於檢查 ErrNoRows
	"testing"      // Go 的測試框架套件
	"time"         // 用於時間比較

	"github.com/andyrestart9/bank/util"   // 自訂的工具包，提供隨機資料產生函式
	"github.com/stretchr/testify/require" // 提供斷言函式，簡化測試中的錯誤檢查
)

// createRandomAccount 會使用隨機資料呼叫 CreateAccount，並驗證資料庫回傳的 Account 欄位是否正確
func createRandomAccount(t *testing.T) Account {
	arg := CreateAccountParams{
		Owner:    util.RandomOwner(),    // 隨機產生一個擁有者名稱（6 個隨機漢字）
		Balance:  util.RandomMoney(),    // 隨機產生一個金額（0～1000）
		Currency: util.RandomCurrency(), // 隨機挑選一種貨幣（USD/EUR/CAD）
	}

	// 呼叫事先在 TestMain 裡建立好連線的 testQueries.CreateAccount，將參數 arg 插入資料庫
	account, err := testQueries.CreateAccount(context.Background(), arg)
	// 確認不會有錯誤發生
	require.NoError(t, err)
	// 確認回傳的 account 結構不為空
	require.NotEmpty(t, account)

	// 驗證資料庫裡的欄位值與我們傳入的一致
	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)

	// 資料庫應該自動填充 ID（流水號）與 CreatedAt（建立時間），因此不能為零值
	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account // 回傳新建的帳戶，以供之後測試使用
}

// TestCreateAccount 單純呼叫 createRandomAccount 並依賴其中的斷言，確保 CreateAccount 沒有 panic 或錯誤
func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

// TestGetAccount 先建立一筆隨機帳戶，然後用 GetAccount 讀取並比對欄位
func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)                                         // 先插入一筆隨機帳戶
	account2, err := testQueries.GetAccount(context.Background(), account1.ID) // 用剛剛回傳的 ID 去查
	require.NoError(t, err)                                                    // 確認 GetAccount 不會錯
	require.NotEmpty(t, account2)                                              // 確認讀到的資料不為空

	// 驗證兩次讀取的欄位都一樣
	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	// CreatedAt 為 timestamp，可能差別很微小，允許一秒以內的誤差
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

// TestUpdateAccount 先建立一筆隨機帳戶，然後呼叫 UpdateAccount 更新 balance，最後驗證只有 balance 變更
func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount(t) // 插入隨機帳戶

	arg := UpdateAccountParams{
		ID:      account1.ID,        // 指定要更新哪一筆帳戶
		Balance: util.RandomMoney(), // 隨機產生新的金額
	}

	account2, err := testQueries.UpdateAccount(context.Background(), arg) // 執行更新
	require.NoError(t, err)                                               // 確認更新不會返回錯誤
	require.NotEmpty(t, account2)                                         // 確認回傳的結構不為空

	// ID、Owner、Currency 都不應該變，只有 Balance 被新值取代
	require.Equal(t, account1.ID, account2.ID)             // ID 不應改變
	require.Equal(t, account1.Owner, account2.Owner)       // Owner 不應改變
	require.Equal(t, arg.Balance, account2.Balance)        // Balance 應該被更新
	require.Equal(t, account1.Currency, account2.Currency) // Currency 不應改變
	// CreatedAt 應該保留原本的建立時間，允許一秒之內誤差
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

// TestDeleteAccount 先建立隨機帳戶，呼叫 DeleteAccount 再用 GetAccount 驗證該筆已不存在
func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)                                  // 插入一筆帳戶
	err := testQueries.DeleteAccount(context.Background(), account1.ID) // 刪除該帳戶
	require.NoError(t, err)                                             // 確認刪除不會有錯誤

	// 再次使用 GetAccount 嘗試查詢已刪除的 ID，應該得到 ErrNoRows
	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.Error(t, err)                             // 確認有錯誤發生
	require.EqualError(t, err, sql.ErrNoRows.Error()) // 錯誤必須是 "no rows in result set"
	require.Empty(t, account2)                        // account2 結構應該是空值
}

// TestListAccounts 建立 10 筆隨機帳戶，然後測試分頁查詢：Limit=5, Offset=5，應該拿到第 6～10 筆
func TestListAccounts(t *testing.T) {
	// 先插入 10 筆隨機帳戶
	for i := 0; i < 10; i++ {
		createRandomAccount(t)
	}

	arg := ListAccountsParams{
		Limit:  5, // 最多取 5 筆
		Offset: 5, // 跳過最前面 5 筆
	}

	accounts, err := testQueries.ListAccounts(context.Background(), arg) // 執行分頁查詢
	require.NoError(t, err)                                              // 確認沒有錯誤
	require.Len(t, accounts, 5)                                          // 回傳切片長度必須是 5

	// 確認每一筆都不是空結構
	for _, account := range accounts {
		require.NotEmpty(t, account)
	}
}
