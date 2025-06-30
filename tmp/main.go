package main

import (
	"fmt"
	"runtime"
	"sync"
)

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
