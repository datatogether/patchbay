package main

import (
	"encoding/json"
	"fmt"
	// "github.com/datatogether/task-mgmt/tasks"
	"github.com/garyburd/redigo/redis"
	"net"
	"time"
)

// Main redis connection
var rconn redis.Conn

func connectRedis() (err error) {
	var netConn net.Conn

	if cfg.RedisUrl == "" {
		return fmt.Errorf("no redis url specified")
	}

	for i := 0; i <= 1000; i++ {
		netConn, err = net.Dial("tcp", cfg.RedisUrl)
		if err != nil {
			return err
			time.Sleep(time.Second)
			continue
		}
		break
	}

	if netConn == nil {
		return fmt.Errorf("no net connection after 1000 tries")
	}

	rconn = redis.NewConn(netConn, time.Second*20, time.Second*20)
	return SubscribeTaskProgress(rconn)
}

func SubscribeTaskProgress(c redis.Conn) error {
	psc := redis.PubSubConn{Conn: c}
	if err := psc.PSubscribe("tasks.*.progress"); err != nil {
		return err
	}
	go func() {
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				fmt.Printf("%s: message: %s\n", v.Channel, v.Data)

				// TODO - other types of messages will eventually come through
				// here...

				res := &ClientResponse{
					Type:      "TASK_PROGRESS",
					RequestId: "server",
					Data:      json.RawMessage(v.Data),
				}

				data, err := json.Marshal(res)
				if err != nil {
					log.Infoln(err.Error())
				} else {
					room.broadcast <- data
				}
			case redis.Subscription:
				fmt.Printf("%s: %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				log.Infoln(v.Error())
			}
		}
	}()
	return nil
}
