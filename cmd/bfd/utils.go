package main

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/tidwall/redcon"
)

func joinCluster(nodeID string, addr string, joinAddr string) error {
	c := redis.NewClient(&redis.Options{
		Addr:         joinAddr,
		MaxRetries:   -1,
		ReadTimeout:  -1,
		WriteTimeout: -1,
	})
	defer c.Close()

	return c.Do(context.TODO(), "join", nodeID, addr).Err()
}

func writeInvalidArgumentsErr(conn redcon.Conn, cmd redcon.Command) {
	conn.WriteError(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Args[0]))
}
