package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/liuc2050/easychat/client"
	"github.com/liuc2050/easychat/server"
	"github.com/liuc2050/easychat/ui"
)

type CmdFunc func([]string) error
type TextFunc func(string) error

type CmdEntry struct {
	Execute CmdFunc
	Send    TextFunc
	Help    string
}

var cmds = map[string]CmdEntry{
	"create": CmdEntry{Execute: createServer, Send: send, Help: "create [[ip][:]port]\t\tstart a server which listens on the local network address."},
	"enter":  CmdEntry{Execute: enterServer, Send: send, Help: "enter [ip:port]\t\tconnect server"},
	"leave":  CmdEntry{Execute: leaveServer, Help: "leave\t\tdisconnect server"},
	"bye":    CmdEntry{Execute: bye, Help: "bye\t\texit program"},
}

var currentCmd string
var srv *server.Server
var cli *client.Client
var shouldExit chan struct{}

type argsErr string

func (e *argsErr) Error() string {
	return string(*e)
}

type WriteFunc func(string)

func (f WriteFunc) Write(p []byte) (n int, err error) {
	s := string(p)
	f(s)
	return len(s), nil
}

var logger = log.New(WriteFunc(ui.Notify), "", log.LstdFlags)

var fileName = flag.String("log", "", "log file name")

func main() {
	flag.Parse()
	if len(*fileName) > 0 {
		file, err := os.OpenFile(*fileName, os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		logger = log.New(file, "", log.LstdFlags)
	}
	ui.Init(logger)
	defer ui.Close()
	helpInfo()
	go ui.Draw()
	shouldExit = make(chan struct{})
	for {
		out, isCmd, err := ui.Scan()
		if err != nil {
			ui.Notify(err.Error())
			continue
		}
		if isCmd {
			if err := executeCmd(out); err != nil {
				if e, ok := err.(*argsErr); ok {
					ui.Notify(e.Error())
					continue
				}
				panic(err)
			}
		} else {
			if err := sendMsg(out[0]); err != nil {
				ui.Notify(err.Error())
				continue
			}
		}

		select {
		case <-shouldExit:
			return
		default:
			//do nothing
		}
	}
}

func helpInfo() {
	ui.Notify("A vim-style chatting program. At insert mode you can type message to send. At last-line mode you can type these commands:")
	for _, v := range cmds {
		ui.Notify(v.Help)
	}
}

func executeCmd(args []string) error {
	if len(args) <= 0 {
		return errors.New("executeCmd: args is empty")
	}

	cmdEntry, ok := cmds[args[0]]
	if !ok || cmdEntry.Execute == nil {
		s := fmt.Sprintf("executeCmd: invalid command[%s]", args[0])
		return (*argsErr)(&s)
	}

	err := cmdEntry.Execute(args)
	if err == nil {
		currentCmd = args[0]
	}

	return err
}

func sendMsg(msg string) error {
	cmdEntry, ok := cmds[currentCmd]
	if !ok || cmdEntry.Send == nil {
		return fmt.Errorf("sendMsg: current command[%s] does not support sending message",
			currentCmd)
	}

	return cmdEntry.Send(msg)
}

func createServer(args []string) error {
	if len(args) != 2 {
		s := "createServer: len(args) should be 2"
		return (*argsErr)(&s)
	}
	srv = server.New(args[1], logger)
	err := srv.Start()
	if err != nil {
		return err
	}
	ui.Notify(fmt.Sprintf("server[%s] is listening.", args[1]))
	cli = client.New("localhost:"+args[1], logger, ui.Notify)
	return cli.EnterServer()
}

func enterServer(args []string) error {
	if len(args) != 2 {
		s := "enterServer: len(args) should be 2"
		return (*argsErr)(&s)
	}
	cli = client.New(args[1], logger, ui.Notify)
	return cli.EnterServer()
}

func send(msg string) error {
	if cli == nil {
		return errors.New("send: cli is nil")
	}
	return cli.Send(msg)
}

func leaveServer(args []string) error {
	if cli == nil {
		return nil
	}
	if err := cli.LeaveServer(); err != nil {
		return err
	}
	cli = nil
	if srv != nil {
		srv.ShutDown()
		srv = nil
	}
	return nil
}

func bye(args []string) error {
	if err := leaveServer(args); err != nil {
		return err
	}
	close(shouldExit)
	return nil
}
