package app

import (
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func CheckConnection(ip string, port int, t int, network string) error {
	var (
		conn net.Conn
		err  error
	)

	for i := 0; ; i++ {
		if network == "tcp6" {
			ip = "[" + ip + "]"
		}
		conn, err = net.DialTimeout(network, ip+":"+strconv.Itoa(port), time.Second*time.Duration(t))
		if err != nil {
			log.Infof("%v attempt retry.. (%d/3)", err, i+1)
			if i == 2 {
				return err
			}
			time.Sleep(time.Second * 5)
		} else {
			conn.Close()
			return nil
		}
	}
}
