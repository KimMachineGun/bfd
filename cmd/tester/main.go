package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
	startedAt := time.Now()
	wg := sync.WaitGroup{}
	ctx := context.Background()
	c := redis.NewClient(&redis.Options{
		Addr:         "localhost:8001",
		MaxRetries:   -1,
		ReadTimeout:  -1,
		WriteTimeout: -1,
		PoolSize:     100,
	})

	cmd := redis.NewBoolCmd(ctx, "set", "hello")
	_ = c.Process(ctx, cmd)
	fmt.Println(cmd.Result())

	keys := make(chan int)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for key := range keys {
				cmd := redis.NewBoolCmd(ctx, "set", fmt.Sprintf("100_%d", key))
				_ = c.Process(ctx, cmd)
				if cmd.Err() != nil {
					log.Fatalln(cmd.Err())
				}
			}
		}()
	}

	for i := 0; i < 10000; i++ {
		keys <- i
	}
	close(keys)

	wg.Wait()
	fmt.Println("end:", time.Now().Sub(startedAt))
}
