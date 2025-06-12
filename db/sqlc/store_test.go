package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	// 寫資料庫交易（database transaction）時一定要非常謹慎。不小心處理多線程（並發，concurrency）的話會有同時讀寫資料庫的情況，發現潛在的鎖定（locking）或資料競爭（race condition）問題
	// 為了確保交易邏輯在多個程式同時執行的情況下仍然正確運作，最好的做法是用多個並行的 Go goroutine（即同時啟動多個 Go 程式執行緒）去跑測試。
	// run n concurrent transfer transactions
	n := 10
	// n := 3
	amount := int64(10)

	errs := make(chan error)
	results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {
		// txName := fmt.Sprintf("tx %d", i+1) // debug
		go func() {
			// ctx := context.WithValue(context.Background(), txKey, txName) // debug
			ctx := context.Background()
			result, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})

			// ** 不要在這裡做驗證 **

			// 如果你在測試中，從主測試程式（也就是那個跑 TestTransferTx 的 goroutine）另外再啟動一個 goroutine 去執行某個函式（比如同時模擬多個併發呼叫），那麼你就 不能直接在那個子 goroutine 裡面用 require （testify/assert）去斷言。

			// require 會呼叫 t.FailNow() 來終止測試
			// require 這類斷言函式在斷言失敗時，底層會呼叫 testing.T.FailNow()（或者 Fatalf），它會馬上結束「呼叫它的那個 goroutine」並且把這個測試標記為失敗。

			// 但如果你是在另一個 goroutine 裡面呼叫 require，它只會終止那個單獨的 goroutine，而不會中斷主測試 goroutine
			// 換句話說，假設你在一條新開的 goroutine 裡面做了 require.NoError(err)，如果 err 不為 nil，那條子 goroutine 會被 “Kill” 掉，但主測試流程還是繼續向下跑，並不會因為子 goroutine 的斷言失敗就把整個 TestTransferTx 給停掉。結果就可能是：你覺得某個條件不滿足應該直接 fail，但測試卻依然跑下去，最終可能把錯誤埋起來，看不到真正的失敗原因。

			// 正確做法：把錯誤和結果傳回主 goroutine，再由主 goroutine 來做斷言
			// 因此，最保險的方法是：在子 goroutine 裡只做業務邏輯的執行，然後把它執行後的 error 或者結果（result）送回給主測試 goroutine（通常用 channel）。主 goroutine 收到後，再做 require.NoError(err)、require.Equal(expected, got) 這類斷言。這樣一旦斷言失敗，主測試就會被立即中斷並報錯，確保測試結果能正確反映問題。

			// 把子 goroutine 執行後的 error 或者結果（result）送回給主 goroutine ，最簡單又安全的方式就是用 Go 語言內建的「通道」（channel），Channel is designed to connect concurrent Go routines,and allow them to safely share data with each other without explicit locking.

			errs <- err
			results <- result
		}()
	}

	// check results
	existed := make(map[int]bool)

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		// check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		// 因為 Queries object 匿名 embedded inside the Store 所以可以直接使用 store.Queries.GetTransfer
		// _, err = store.Queries.GetTransfer(context.Background(), transfer.ID)
		_, err = store.Queries.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// check entries
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = store.Queries.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.Queries.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// check accounts
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, account1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)

		// check accounts' balance
		fmt.Println(">> tx:", fromAccount.Balance, toAccount.Balance)
		diff1 := account1.Balance - fromAccount.Balance
		diff2 := toAccount.Balance - account2.Balance
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		require.True(t, diff1%amount == 0) // 1 * amount, 2 * amount, 3 * amount, ... n * amount

		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// check the final updated balance
	updatedAccount1, err := store.Queries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	updatedAccount2, err := store.Queries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)
	require.Equal(t, account1.Balance-int64(n)*amount, updatedAccount1.Balance)
	require.Equal(t, account2.Balance+int64(n)*amount, updatedAccount2.Balance)
}

func TestTransferTxDeadlock(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	n := 10
	amount := int64(10)

	errs := make(chan error)

	// run n concurrent transfer transactions
	for i := 0; i < n; i++ {
		fromAccountID := account1.ID
		toAccountID := account2.ID

		if i%2 == 1 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}

		go func() {
			ctx := context.Background()
			_, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			errs <- err
		}()
	}

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}

	// check the final updated balance
	updatedAccount1, err := store.Queries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	updatedAccount2, err := store.Queries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)
	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}
