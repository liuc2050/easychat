package client

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/liuc2050/easychat/util"
)

type Client struct {
	noCopy util.NoCopy

	srvAddr string
	conn    net.Conn
	onRead  func(string)
	logger  *log.Logger
	wg      *sync.WaitGroup
}

func New(srvAddr string, l *log.Logger, onRead func(string)) *Client {
	return &Client{srvAddr: srvAddr, onRead: onRead, logger: l, wg: new(sync.WaitGroup)}
}

func (cli *Client) EnterServer() error {
	if cli == nil {
		return errors.New("EnterServer: cli is nil")
	}

	var err error
	cli.conn, err = net.Dial("tcp", cli.srvAddr)
	if err == nil && cli.onRead != nil {
		cli.wg.Add(1)
		go func() {
			defer cli.wg.Done()
			scanner := bufio.NewScanner(cli.conn)
			for {
				if scanner.Scan() {
					cli.onRead(scanner.Text())
				} else {
					cli.logger.Printf("Read error:%s", scanner.Err())
					break
				}
			}
		}()
	}
	return err
}

func (cli *Client) LeaveServer() error {
	if cli == nil {
		return errors.New("LeaveServer: cli is nil")
	}
	if cli.conn == nil {
		return errors.New("LeaveServer: cli.conn is nil")
	}
	if err := cli.conn.Close(); err != nil {
		return err
	}
	cli.wg.Wait()
	return nil
}

func (cli *Client) Send(msg string) error {
	if cli == nil {
		return errors.New("Send: cli is nil")
	}
	if cli.conn == nil {
		return errors.New("Send: cli.conn is nil")
	}
	_, err := fmt.Fprintln(cli.conn, msg)
	return err
}
