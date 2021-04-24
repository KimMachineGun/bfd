package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/tidwall/redcon"

	"github.com/KimMachineGun/bloomfilterd/internal/store"

	"github.com/KimMachineGun/flago"

	_ "net/http/pprof"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	config := DefaultConfig

	err := flago.Bind(flag.CommandLine, &config)
	if err != nil {
		log.Fatalln(err)
	}
	flag.Parse()

	err = config.Validate()
	if err != nil {
		log.Fatalln(err)
	}

	s, err := store.New(&store.Config{
		N:         config.N,
		P:         config.P,
		Retention: config.Retention,

		NodeID:           config.NodeID,
		Addr:             config.Addr,
		TransportMaxPool: config.TransportMaxPool,
		TransportTimeout: config.TransportTimeout,
		Dir:              config.Dir,
		SnapshotRetain:   config.SnapshotRetain,

		ApplyTimeout: config.ApplyTimeout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	if config.JoinAddr == "" {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
		err = s.Bootstrap()
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		err = joinCluster(config.NodeID, config.Addr, config.JoinAddr)
		if err != nil {
			log.Fatalln(err)
		}
	}

	err = redcon.ListenAndServe(config.HTTP,
		func(conn redcon.Conn, cmd redcon.Command) {
			switch strings.ToLower(string(cmd.Args[0])) {
			default:
				conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
			case "ping":
				conn.WriteString("PONG")
			case "join":
				if len(cmd.Args) != 3 {
					writeInvalidArgumentsErr(conn, cmd)
					return
				}

				nodeID, addr := cmd.Args[1], cmd.Args[2]
				err = s.Join(string(nodeID), string(addr))
				if err != nil {
					conn.WriteError(err.Error())
				}
			case "snapshot":
				if len(cmd.Args) != 1 {
					writeInvalidArgumentsErr(conn, cmd)
					return
				}

				nodeID, addr := cmd.Args[1], cmd.Args[2]
				err = s.Join(string(nodeID), string(addr))
				if err != nil {
					conn.WriteError(err.Error())
				}
			case "set":
				if len(cmd.Args) != 2 {
					writeInvalidArgumentsErr(conn, cmd)
					return
				}

				ok, err := s.Set(string(cmd.Args[1]))
				if err != nil {
					conn.WriteError(err.Error())
					return
				}
				if !ok {
					conn.WriteInt(0)
				} else {
					conn.WriteInt(1)
				}
			case "get":
				if len(cmd.Args) != 2 {
					writeInvalidArgumentsErr(conn, cmd)
					return
				}

				ok, err := s.Check(string(cmd.Args[0]))
				if err != nil {
					conn.WriteError(err.Error())
					return
				}
				if !ok {
					conn.WriteInt(0)
				} else {
					conn.WriteInt(1)
				}
			}
		},
		func(conn redcon.Conn) bool {
			// Use this function to accept or deny the connection.
			// log.Printf("accept: %s", conn.RemoteAddr())
			return true
		},
		func(conn redcon.Conn, err error) {
			// This is called when the connection has been closed
			// log.Printf("closed: %s, err: %v", conn.RemoteAddr(), err)
		},
	)
	if err != nil {
		log.Fatalln(err)
	}
}
