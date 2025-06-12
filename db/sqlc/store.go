package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides all functions to execute database transactions
type Store struct {
	*Queries
	db *sql.DB
}

// NewStore 就是给外部用的“构造函数”：
// 一次性把你的数据库连接 *sql.DB 和 sqlc 生成的 *Queries 包装到一个 Store 实例
// 把 *sql.DB + *Queries 包成一个 Store 的好处主要有三点：
// 统一依赖注入：
// 在 handler、service 里只要拿到一个 Store，不必同时传 *sql.DB 和 *Queries，用起来更简洁，也更方便在测试里替换成 mock。
// 封装事务和业务逻辑：
// Queries 只是对单表的 CRUD，没法一气儿执行业务流程（像转账要同时更新转账表、流水表、账目表）。Store 里可以写 TransferTx 这种跨多张表、要开/提交/回滚事务的组合操作，让业务代码只管调用，不用自己管事务。
// 隐藏实现细节、便于扩展：
// 以后要加日志／监控／重试／读写分离，只要改 NewStore 或 Store 方法，业务层不需要动。把数据库层的变化都封装在 Store 里，层次更清晰、可维护性更好。

// 為什麼 Queries 没法执行业务流程， Store 里可以写跨多张表、要开/提交/回滚事务的组合操作？
// Queries 是由 sqlc 自动生成的一组「一对一」方法，它们各自只负责执行单条 SQL，不包含事务的开启、提交或回滚，也无法把多条操作合并到同一个事务里。每次调用时，都要你手动决定传入的是全局 *sql.DB（不在事务中）还是某个 *sql.Tx（在事务中），它自己不会管理生命周期。
// 舉例： Store 则在 execTx 里：
// 调用 db.BeginTx(ctx,…) 开启事务，
// 用 New(tx) 得到一组绑定在同一事务 tx 上的 Queries，
// 在传入的 fn 中连续调用多条 q.* 方法，
// 最后根据 fn 返回的 error 自动 Commit 或 Rollback。
// 这样，你就能把跨多张表、多个 CRUD 操作的业务流程，包装在同一个事务里，保证要么全成功要么全回滚，而不是把这些关键信息散落到单个 Queries 方法里去管理。

// NewStore creates a new Store instance
func NewStore(db *sql.DB) *Store {
	// 怎麼確認 *sql.DB 實例實現了 DBTX interface？
	// 要在代码里确保无误、并让其他读代码的人也一看就懂，var _ Interface = (*Type)(nil) 就是最简洁、最惯用的做法
	// (*sql.DB)(nil)——“把 nil 转成 *sql.DB 类型”
	// var _ DBTX = …——把这个 nil 指针赋给接口类型 DBTX，编译器就会在这一行做类型检查：如果 *sql.DB 没有实现 DBTX 中的所有方法，就报编译错误。
	var _ DBTX = (*sql.DB)(nil) // 確認 *sql.DB 實例實現了 DBTX interface

	// 創建一個新的 SQLStore 實例，並返回一個 Store 實例
	return &Store{
		db: db,
		// func New(db DBTX) *Queries 接收一個 DBTX type 參數，因為 db 是 *sql.DB 實例， *sql.DB 實現了 DBTX interface，所以可以傳入 db 參數
		Queries: New(db),
	}
}

// 為什麼這邊需要 ctx context.Context ？
// 因为我们要在事务里对数据库做 I/O 操作，而 Go 里所有带上下文（取消、超时、trace、元数据）需求的外部调用，都约定要传一个 context.Context：
// store.db.BeginTx(ctx, nil) 本身就需要 ctx，用来控制事务的取消或超时。
// 事务里后续的 q.CreateTransfer(ctx, …)、q.CreateEntry(ctx, …) 等方法底层会调用 ExecContext/QueryContext，也都要拿这个 ctx 去判断是不是该终止操作。
// 把 ctx 一直往下传可以让上层（比如 HTTP 请求）一旦超时或被取消，就能自动中断数据库事务，避免资源泄漏。
// 所以，传 ctx 是 Go 在所有数据库／网络／I/O 路径上统一的最佳实践，也是 BeginTx、ExecContext 一类 API 的要求。

// 為什麼這邊需要 fn func(*Queries) error ？
// 这里用 fn func(*Queries) error 是为了把 “在同一个事务里执行一系列查询” 的逻辑抽象出来：
// execTx 负责打开事务（BeginTx）、在结束时根据 fn 返回的 error 决定是 Rollback 还是 Commit，把事务的生命周期管理（开启/回滚/提交）都封装好了。
// fn 就是让你传入一段“具体要在这次事务里做什么”的业务代码，它接收一个 *Queries（内部其实是绑定在这个事务 tx 上的），你在里面可以任意调用 q.CreateTransfer(ctx,…)、q.UpdateAccount(ctx,…) 等操作。
// 这样做的好处是：
// 把事务的模板代码（Begin/Commit/Rollback）统一放在 execTx 里，业务逻辑里只管写自己的 fn，不必每次都重复写错误处理和回滚代码。
// 保证所有通过 q := New(tx) 得到的查询都在同一个事务上下文里执行。
// 简而言之，fn func(*Queries) error 就是把“要干什么”这段代码注入到事务管理器里，让 execTx 帮你管好开关和异常处理。

// execTx executes a function within a database transaction
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var _ DBTX = (*sql.Tx)(nil) // 確認 *sql.Tx 實例實現了 DBTX interface
	// 如果寫成 q := New(store.db) 的話
	// 那么接下来对 q 的所有操作都会直接走全局连接池，根本不会进入你的事务 tx。也就是说：
	// 你的 BeginTx/Rollback/Commit 只对那条空事务起作用，业务逻辑已经在外面执行并且立刻提交了。
	// 一旦中途出错，tx.Rollback() 也不会撤销那些已经跑在 *sql.DB 上的操作，失去原子性保证。
	// 虽然它“不报错”，但你会丢失事务的语义。为了确保一组读写要么全成功要么全回滚，一定要用 New(tx)。
	// 所以在 execTx 里要用 New(tx)，才能把后续的所有操作都“绑在”这个事务里，保证 fn(q) 里所有的 q.* 调用都在同一个事务上下文执行，做到原子性。
	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// 这些 json:"..." 标签是给 Go 的 encoding/json（或者其他兼容 JSON tag 的库）用的，作用就是：
// 指定字段在序列化（Marshal）成 JSON 时用什么键名，反序列化（Unmarshal）时对应哪个字段。
// 例如 FromAccountID int64 \json:"from_account_id"\``，输出的 JSON 字段就是 "from_account_id"，而不是默认的 "FromAccountID"。
// 保证你的 HTTP 请求体／响应体和前端约定的 snake_case（或其他规范）保持一致。

// TransferTxParams contains the input parameters of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// define a named type for context key to avoid collision
// type txKeyType struct{} // debug

// txKey used as context key for transaction name
// var txKey = txKeyType{} // debug

// TransferTx performs a money transfer from one account to the other.
// It creates the transfer, add account entries, and update accounts' balance within a single database transaction
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// txName := ctx.Value(txKey) // debug

		// fmt.Println(txName, "create transfer") // debug
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		// fmt.Println(txName, "create entry 1") // debug
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		// fmt.Println(txName, "create entry 2") // debug
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// 不用 SELECT … FOR NO KEY UPDATE ，先 SELECT 再 UPDATE，事务稍复杂
		// // fmt.Println(txName, "get account 1") // debug
		// account1, err := q.GetAccountForUpdate(ctx, arg.FromAccountID)
		// if err != nil {
		// 	return err
		// }

		// // fmt.Println(txName, "update account 1") // debug
		// result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
		// 	ID:      arg.FromAccountID,
		// 	Balance: account1.Balance - arg.Amount,
		// })
		// if err != nil {
		// 	return err
		// }

		// // fmt.Println(txName, "get account 2") // debug
		// account2, err := q.GetAccountForUpdate(ctx, arg.ToAccountID)
		// if err != nil {
		// 	return err
		// }

		// // fmt.Println(txName, "update account 2") // debug
		// result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
		// 	ID:      arg.ToAccountID,
		// 	Balance: account2.Balance + arg.Amount,
		// })
		// if err != nil {
		// 	return err
		// }

		// 改用 UPDATE … RETURNING ，一步原子：只要更新并拿回新值，只会使用行级排他锁，不会牵扯 transaction-ID 锁，最不易死锁
		// 固定鎖序：先更新較小的 id、再更新較大的 id（或固定其它排序），所有程式遵守同一順序，就不會交叉等待。
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
			if err != nil {
				return err
			}
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return result, err
}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		// 因為我們用 named return variable (account1 Account, account2 Account, err error) ，所以可以直接 return
		// 這邊直接 return 等價於 return account1, account2, err
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}
