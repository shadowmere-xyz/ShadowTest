package ssproxy

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/socks"
	log "github.com/sirupsen/logrus"
)

/*
Taken from https://github.com/shadowsocks/go-shadowsocks2/blob/master/tcp.go since the original
code is not importable and needed some modifications to accept only one connection.
*/

// ListenForOneConnection create a local socks5 proxy and listen for 1 connection
func ListenForOneConnection(addr, server string, shadow func(net.Conn) net.Conn, ready chan bool, getAddr func(net.Conn) (socks.Addr, error)) {
	l, err := net.Listen("tcp", addr)
	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			log.Errorf("failed to close connection: %v", err)
		}
	}(l)
	if err != nil {
		log.Errorf("failed to listen on %s: %v", addr, err)
		return
	}
	ready <- true

	c, err := l.Accept()
	if err != nil {
		log.Errorf("failed to accept: %s", err)
		return
	}

	go func() {
		defer func(c net.Conn) {
			err := c.Close()
			if err != nil {
				log.Errorf("failed to close connection: %v", err)
			}
		}(c)
		tgt, err := getAddr(c)
		if err != nil {
			// UDP: keep the connection until disconnect then free the UDP socket
			if errors.Is(err, socks.InfoUDPAssociate) {
				buf := make([]byte, 1)
				// block here
				for {
					_, err := c.Read(buf)
					var neterr net.Error
					if errors.As(err, &neterr) && neterr.Timeout() {
						log.Infof("connection timed out")
						continue
					}
					log.Info("UDP Associate End.")
					return
				}
			}

			log.Errorf("failed to get target address: %v", err)
			return
		}

		rc, err := net.Dial("tcp", server)
		if err != nil {
			log.Warnf("failed to connect to server %v: %v", server, err)
			return
		}
		defer func(rc net.Conn) {
			err := rc.Close()
			if err != nil {
				log.Errorf("failed to close connection to server %v: %v", server, err)
			}
		}(rc)
		rc = shadow(rc)

		if _, err = rc.Write(tgt); err != nil {
			log.Warnf("failed to send target address: %v", err)
			return
		}

		log.Infof("proxy %s <-> %s <-> %s", c.RemoteAddr(), server, tgt)
		if err = relay(rc, c); err != nil {
			log.Warnf("relay error: %v", err)
		}
	}()
}

// relay copies between left and right bidirectionally
func relay(left, right net.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	doneCh := make(chan struct{})

	copyConn := func(dst, src net.Conn, name string) {
		defer wg.Done()
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				errDst := dst.SetReadDeadline(time.Now())
				if errDst != nil {
					log.Errorf("failed to set read deadline: %v", errDst)
				}
				errSrc := src.SetReadDeadline(time.Now())
				if errSrc != nil {
					log.Errorf("failed to set read deadline: %v", errSrc)
				}
			case <-done:
			}
		}()

		_, err := io.Copy(dst, src)
		close(done)
		cancel()

		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			errCh <- err
		}
	}

	wg.Add(2)
	go copyConn(right, left, "left->right")
	go copyConn(left, right, "right->left")

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}
