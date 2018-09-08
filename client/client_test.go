package client

import (
	"net"
	"testing"
)

func TestNew(t *testing.T) {
	addr := "2395"
	cli := New(addr)

	if cli == nil {
		t.Errorf("cli should not be nil")
	}

	if cli.srvAddr != addr {
		t.Errorf("got %s, want %s", cli.srvAddr, addr)
	}
}

func TestEnterServer(t *testing.T) {
	var cli *Client
	if err := cli.EnterServer(); err == nil {
		t.Errorf("when cli nil, it should return error")
	}
	cli = New("localhost:2048")
	if err := cli.EnterServer(); err == nil {
		t.Errorf("server not start, should return error")
	}
	if cli.conn != nil {
		t.Errorf("got conn not nil, want nil")
	}

	ln, err := net.Listen("tcp", ":2048")
	if err != nil {
		t.Fatalf("server start failed.%v", err)
	}
	defer ln.Close()
	go func() {
		if _, err := ln.Accept(); err != nil {
			t.Fatalf("accept err: %v", err)
		}
	}()
	if err := cli.EnterServer(); err != nil {
		t.Fatalf("EneterServer got %v, want nil", err)
	}
	if cli.conn == nil {
		t.Errorf("got conn nil")
	}
	cli.LeaveServer()
}

func TestLeaveServer(t *testing.T) {
	var cli *Client
	if err := cli.LeaveServer(); err == nil {
		t.Errorf("when cli nil, it should return error")
	}

	cli = New("localhost:2048")
	if err := cli.LeaveServer(); err == nil {
		t.Errorf("when cli.conn nil, it should return error")
	}

	ln, _ := net.Listen("tcp", ":2048")
	stopCh := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("accept err: %v", err)
		}
		select {
		case <-stopCh:
			conn.Close()
		}
	}()
	cli.EnterServer()
	if err := cli.LeaveServer(); err != nil {
		t.Errorf("got [%v], want nil", err)
	}
	close(stopCh)
}

func TestSend(t *testing.T) {
	var cli *Client
	if err := cli.Send("dkkd"); err == nil {
		t.Errorf("when cli nil, should return error")
	}
	cli = New("localhost:3047")
	if err := cli.Send("跨学科"); err == nil {
		t.Errorf("got nil , want error")
	}

	ln, _ := net.Listen("tcp", ":3047")
	stopCh := make(chan struct{})
	msg := "可哦哦巍峨"
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("accept err: %v", err)
		}
		var buf [2048]byte
		n, _ := conn.Read(buf[:])
		s := string(buf[:n])
		if s != msg+"\n" {
			t.Errorf("got %s", s)
		}
		select {
		case <-stopCh:
			conn.Close()
		}
	}()
	cli.EnterServer()
	if err := cli.Send(msg); err != nil {
		t.Errorf("got error[%v], want nil", err)
	}
	if err := cli.LeaveServer(); err != nil {
		t.Errorf("got error[%v], want nil", err)
	}
	close(stopCh)
}
