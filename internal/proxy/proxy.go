package proxy

import (
	"fmt"
	"io"
	"net"
)

func ToAddr(hostname, port string) string {
	return fmt.Sprintf("%s:%s", hostname, port)
}

func Proxy(addr1, addr2 string) error {
	l, err := net.Listen("tcp", addr1)
	if err != nil {
		return err
	}

	defer func() {
		logCloser("closing net.Listen", l.Close())
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go func(c net.Conn) {
			defer func() {
				logCloser("closing host conn", c.Close())
			}()
			pconn, err := net.Dial("tcp", addr2)
			if err != nil {
				return
			}
			defer func() {
				logCloser("closing addr2 conn", pconn.Close())
			}()
			go func() {
				_, err := io.Copy(pconn, c)
				if err != nil {
					fmt.Println("error writing to ctr", err)
				}
			}()
			_, err = io.Copy(c, pconn)
			if err != nil {
				fmt.Println("error writing back to host", err)
			}
		}(conn)
	}
}

func logCloser(what string, err error) {
	if err != nil {
		fmt.Printf("%s error: %s\n", what, err)
	}
}
