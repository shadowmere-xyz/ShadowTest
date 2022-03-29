package ssproxy

import (
	"errors"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

/*
Taken from https://github.com/shadowsocks/go-shadowsocks2/blob/master/tcp.go since the original
code is not importable and needed some modifications to accept only one connection.
*/

func ListenForOneConnection(addr, server string, shadow func(net.Conn) net.Conn, ready chan bool, getAddr func(net.Conn) (socks.Addr, error)) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("failed to listen on %s: %v", addr, err)
		return
	}
	ready <- true

	c, err := l.Accept()
	if err != nil {
		log.Printf("failed to accept: %s", err)
		return
	}

	go func() {
		defer c.Close()
		tgt, err := getAddr(c)
		if err != nil {

			// UDP: keep the connection until disconnect then free the UDP socket
			if err == socks.InfoUDPAssociate {
				buf := make([]byte, 1)
				// block here
				for {
					_, err := c.Read(buf)
					if err, ok := err.(net.Error); ok && err.Timeout() {
						continue
					}
					log.Printf("UDP Associate End.")
					return
				}
			}

			log.Printf("failed to get target address: %v", err)
			return
		}

		rc, err := net.Dial("tcp", server)
		if err != nil {
			log.Printf("failed to connect to server %v: %v", server, err)
			return
		}
		defer rc.Close()
		rc = shadow(rc)

		if _, err = rc.Write(tgt); err != nil {
			log.Printf("failed to send target address: %v", err)
			return
		}

		log.Printf("proxy %s <-> %s <-> %s", c.RemoteAddr(), server, tgt)
		if err = relay(rc, c); err != nil {
			log.Printf("relay error: %v", err)
		}
	}()
}

// relay copies between left and right bidirectionally
func relay(left, right net.Conn) error {
	var err, err1 error
	var wg sync.WaitGroup
	var wait = 5 * time.Second
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err1 = io.Copy(right, left)
		right.SetReadDeadline(time.Now().Add(wait)) // unblock read on right
	}()
	_, err = io.Copy(left, right)
	left.SetReadDeadline(time.Now().Add(wait)) // unblock read on left
	wg.Wait()
	if err1 != nil && !errors.Is(err1, os.ErrDeadlineExceeded) { // requires Go 1.15+
		return err1
	}
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		return err
	}
	return nil
}
