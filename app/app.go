package app

import (
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func CheckConnection(addr string, port int, t int) error {
	var (
		conn net.Conn
		err  error
	)

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	ips := LookupIP(addr)
	count := 0
loop:
	for i := range ips {
		// only check ipv4 address
		conn, err = net.DialTimeout("tcp4", ips[i]+":"+strconv.Itoa(port), time.Second*time.Duration(t))
		if err != nil {
			count++
			if count > 3 {
				return err
			}
			log.Infof("%v attempt retry.. (%d/3)", err, count)
			time.Sleep(time.Second * 5)
			goto loop
		} else {
			return nil
		}
	}

	return err
}
