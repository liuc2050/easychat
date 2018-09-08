package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

var std = log.New(os.Stderr, "", log.LstdFlags)

func TestStart(t *testing.T) {
	srv := New("4321", std)
	if err := srv.Start(); err != nil {
		t.Fatalf("start failed:%v", err)
	}
	if srv.ln == nil {
		t.Fatalf("srv.ln got nil")
	}
	select {
	case <-srv.stopper1.StopCh:
		t.Fatalf("srv.stopper1 should block")
	case <-srv.stopper2.StopCh:
		t.Fatalf("srv.stopper2 should block")
	default:
		//do nothing
	}
	srv.ln.Close()
	srv.stopper1.Stop()
	srv.stopper2.Stop()
}

func TestBroadcast(t *testing.T) {
	srv := New("3829", std)
	srv.stopper1.N.Add(1)
	go srv.broadcast(srv.stopper1)
	cli1 := make(chan string)
	cli2 := make(chan string)
	srv.entering <- cli1
	srv.entering <- cli2
	srv.messages <- "cli entered"
	select {
	case msg := <-cli1:
		if msg != "cli entered" {
			t.Fatalf("cli should receive message")
		}
	case msg := <-cli2:
		if msg != "cli entered" {
			t.Fatalf("cli should receive message")
		}
	}

	srv.leaving <- cli1
	select {
	case _, ok := <-cli1:
		if ok {
			t.Fatalf("cli1 should be closed")
		}
	}

	srv.messages <- "cli left"
	select {
	case msg := <-cli2:
		if msg != "cli left" {
			t.Fatalf("cli2 should receive message")
		}
	}

	srv.stopper1.Stop()
	select {
	case _, ok := <-cli2:
		if ok {
			t.Fatalf("cli2 should be closed")
		}
	}
}

func TestHandleConn(t *testing.T) {
	srv := New("3829", std)
	var err error
	srv.ln, err = net.Listen("tcp", ":"+srv.port)
	if err != nil {
		t.Fatalf("srv listen failed:%v", err)
	}
	defer srv.ln.Close()
	go func() {
		for i := 0; i < 2; i++ {
			conn, err := srv.ln.Accept()
			if err != nil {
				t.Fatalf("acept error:%v", err)
			}
			srv.stopper1.N.Add(1)
			srv.handleConn(conn, srv.stopper1)
		}
	}()

	conn, err := net.Dial("tcp", "localhost:"+srv.port)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()
	select {
	case msg := <-srv.messages:
		fmt.Println(msg)
	case <-time.After(2 * time.Second):
		t.Fatalf("srv.messages does not receive message")
	}
	var cli client
	select {
	case cli = <-srv.entering:
		//do nothing
	case <-time.After(2 * time.Second):
		t.Fatalf("srv.entering does not receive message")
	}

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString("你好" + "\n")
	if err != nil {
		t.Fatalf("write error:%v", err)
	}
	writer.Flush()
	select {
	case msg := <-srv.messages:
		if strings.Index(msg, "你好") < 0 {
			t.Fatalf("srv.messages should receive messsage")
		}
	}

	cli <- "你好"
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() && scanner.Err() != nil {
		t.Fatalf("scan error :%v", scanner.Err())
	}
	if scanner.Text() != "你好" {
		t.Fatalf("scnner.Text() does not correct")
	}

	conn.Close()
	select {
	case msg := <-srv.messages:
		fmt.Println(msg)
	case <-time.After(2 * time.Second):
		t.Fatalf("srv.messages recv timeout")
	}
	select {
	case c := <-srv.leaving:
		if c != cli {
			t.Fatalf("srv.leaving did not recv cli")
		}
	}

	conn2, err := net.Dial("tcp", "localhost:"+srv.port)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn2.Close()
	var cli2 client
	select {
	case cli2 = <-srv.entering:
		//do nothing
	}
	close(srv.stopper1.StopCh)
	select {
	case c := <-srv.leaving:
		if c != cli2 {
			t.Fatalf("srv.leaving did not recv cli2")
		}
	}
	srv.stopper1.N.Wait()
}

func TestShutDown(t *testing.T) {
	var srv *Server
	srv.ShutDown()
	srv = New("2344", std)
	srv.ShutDown()
	if err := srv.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	conn1, err := net.Dial("tcp", "localhost:"+srv.port)
	if err != nil {
		t.Fatalf("conn1 dial error:%v", err)
	}
	defer conn1.Close()
	conn2, err := net.Dial("tcp", "localhost:"+srv.port)
	if err != nil {
		t.Fatalf("conn2 dial error:%v", err)
	}
	defer conn2.Close()
	ch := make(chan bool)
	go func() {
		srv.ShutDown()
		close(ch)
	}()
	select {
	case <-ch:
	case <-time.After(12 * time.Second):
		t.Fatalf("ShutDown time out!")
	}
}
