package app

import (
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func CheckConnection(ip string, port int, t int, network string) (int64, error) {
	var (
		conn net.Conn
		err  error
	)

	for i := 0; ; i++ {
		if network == "tcp6" {
			ip = "[" + ip + "]"
		}
		start := time.Now()
		conn, err = net.DialTimeout(network, ip+":"+strconv.Itoa(port), time.Second*time.Duration(t))
		d := time.Since(start)
		if err != nil {
			log.Infof("%v attempt retry.. (%d/3)", err, i+1)
			if i == 2 {
				return 0, err
			}
			time.Sleep(time.Second * 5)
		} else {
			conn.Close()
			return d.Milliseconds(), nil
		}
	}
}
