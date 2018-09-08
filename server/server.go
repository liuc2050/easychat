package server

import (
	"bufio"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/liuc2050/easychat/util"
)

type Server struct {
	noCopy util.NoCopy

	port               string //监听端口
	ln                 net.Listener
	stopper1, stopper2 *util.Stopper //分阶段的控制结束
	logger             *log.Logger

	entering chan client //client进入通知
	leaving  chan client //client离开通知
	messages chan string //消息广播通道
}

const (
	//各通道cap值
	capEntering int = 10
	capLeaving  int = 1
	capMessages int = 1024
	capClient   int = 100
)

type client chan<- string //只能发送操作（每个客户端消息发送通道）

func New(port string, l *log.Logger) *Server {
	return &Server{port: port,
		stopper1: util.NewStopper(),
		stopper2: util.NewStopper(),
		logger:   l,
		entering: make(chan client, capEntering),
		leaving:  make(chan client, capLeaving),
		messages: make(chan string, capMessages),
	}
}

func (s *Server) Start() error {
	if s == nil {
		return errors.New("Server.Start: s is nil")
	}
	var err error
	s.ln, err = net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}

	s.stopper2.N.Add(1)
	go s.broadcast(s.stopper2) //第二阶段才终止

	s.stopper1.N.Add(1)
	go func() {
		defer s.stopper1.N.Done()
		stopper := util.NewStopper()
		defer stopper.Stop()
		for {
			select {
			case <-s.stopper1.StopCh:
				return
			default:
				conn, err := s.ln.Accept()
				if err != nil {
					s.logger.Printf("Accept error:%v", err)
					return
				}
				stopper.N.Add(1)
				go s.handleConn(conn, stopper)
			}
		}
	}()

	return nil
}

func (s *Server) broadcast(parentStop *util.Stopper) {
	defer parentStop.N.Done()
	clients := make(map[client]*util.Stopper)
	for {
		select {
		case cli := <-s.entering:
			clients[cli] = nil
		case msg := <-s.messages:
			for cli, st := range clients {
				select {
				case cli <- msg:
					//do nothing
				default:
					//阻塞则走异步
					if st == nil {
						st = util.NewStopper()
						clients[cli] = st
					}
					st.N.Add(1)
					go func(cli client, msg string, st *util.Stopper) {
						defer st.N.Done()
						select {
						case cli <- msg:
							//do nothing
						case <-st.StopCh:
							return
						}
					}(cli, msg, st)
				}
			}
		case cli := <-s.leaving:
			//优雅结束
			clients[cli].Stop()
			delete(clients, cli)
			close(cli)
		case <-parentStop.StopCh:
			for cli, st := range clients {
				st.Stop()
				delete(clients, cli)
				close(cli)
			}
			return
		}
	}
}

//goroutine
func (s *Server) handleConn(conn net.Conn, parentStop *util.Stopper) {
	defer parentStop.N.Done()

	ch := make(chan string, capClient)
	s.entering <- ch
	name := conn.RemoteAddr().String()
	//notification
	s.messages <- "[" + name + "]" + " is entering."

	writerStop := make(chan struct{})

	//read
	var n sync.WaitGroup
	defer n.Wait()
	defer conn.Close() //when func returns, it should invoke first

	n.Add(1)
	go func() {
		defer n.Done()
		scanner := bufio.NewScanner(conn)
		for {
			select {
			case <-parentStop.StopCh:
				return
			default:
				if !scanner.Scan() {
					s.logger.Printf("read error:%v", scanner.Err())
					close(writerStop)
					return
				}
				s.messages <- "[" + name + "]: " + scanner.Text()
			}
		}
	}()

loop:
	for { //write
		select {
		case msg := <-ch:
			if _, err := conn.Write([]byte(msg + "\n")); err != nil {
				//写不成功，认为已经离开
				s.logger.Printf("write error:%v", err)
				s.messages <- "[" + name + "]" + " has left."
				break loop
			}
		case <-parentStop.StopCh:
			s.messages <- "[" + name + "]" + " is leaving."
			break loop
		case <-writerStop:
			s.messages <- "[" + name + "]" + " has left."
			break loop
		}
	}
	s.leaving <- ch
}

func (s *Server) ShutDown() {
	if s == nil || s.ln == nil {
		return
	}
	s.ln.Close()
	s.stopper1.Stop()
	//broadcast最后关闭
	s.stopper2.Stop()
}
