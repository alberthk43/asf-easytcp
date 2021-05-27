package server

import (
	"github.com/DarthPestilane/easytcp/logger"
	"github.com/DarthPestilane/easytcp/packet"
	"github.com/DarthPestilane/easytcp/router"
	"github.com/DarthPestilane/easytcp/session"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"syscall"
)

type TcpServer struct {
	rwBufferSize   int
	listener       *net.TCPListener
	listenerClosed chan struct{}
	log            *logrus.Entry
	msgPacker      packet.Packer
	msgCodec       packet.Codec
}

type Option struct {
	RWBufferSize int
	MsgPacker    packet.Packer
	MsgCodec     packet.Codec
}

func NewTcp(opt Option) *TcpServer {
	if opt.MsgPacker == nil {
		opt.MsgPacker = &packet.DefaultPacker{}
	}
	if opt.MsgCodec == nil {
		opt.MsgCodec = &packet.DefaultCodec{}
	}
	return &TcpServer{
		listener:       nil,
		listenerClosed: make(chan struct{}),
		log:            logger.Default.WithField("scope", "tcp_server"),
		rwBufferSize:   opt.RWBufferSize,
		msgPacker:      opt.MsgPacker,
		msgCodec:       opt.MsgCodec,
	}
}

func (t *TcpServer) Serve(addr string) error {
	address, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	lis, err := net.ListenTCP("tcp", address)
	if err != nil {
		return err
	}
	t.listener = lis

	return t.acceptLoop()
}

func (t *TcpServer) acceptLoop() error {
	for {
		select {
		case <-t.listenerClosed:
			return nil // graceful shutdown
		default:
		}
		conn, err := t.listener.AcceptTCP()
		if err != nil {
			t.log.Errorf("accept err: %s", err)
			return err
		}
		if t.rwBufferSize > 0 {
			if err := conn.SetReadBuffer(t.rwBufferSize); err != nil {
				t.log.Errorf("set read buffer err: %s", err)
				return err
			}
			if err := conn.SetWriteBuffer(t.rwBufferSize); err != nil {
				t.log.Errorf("set write buffer err: %s", err)
				return err
			}
		}

		// handle conn in a new goroutine
		go t.handleConn(conn)
	}
}

// handleConn
// create a new session and save it to memory
// read loop
// route incoming message to handler
// write loop
// wait for session to close
// remove session from memory
func (t *TcpServer) handleConn(conn *net.TCPConn) {
	// create a new session
	sess := session.New(conn, t.msgPacker, t.msgCodec)
	session.Sessions().Add(sess)

	// read loop
	go sess.ReadLoop()

	// route incoming message to handler
	go router.Inst().Loop(sess)

	// write loop
	go sess.WriteLoop()

	// wait to close
	if err := sess.WaitToClose(); err != nil {
		t.log.Errorf("session close err: %s", err)
	}
	t.log.Warnf("session closed")

	// sess.Conn has been closed, remove current session
	session.Sessions().Remove(sess.Id)
}

func (t *TcpServer) Stop() error {
	session.Sessions().Range(func(id string, sess *session.Session) (next bool) {
		sess.Close()
		return true
	})
	close(t.listenerClosed)
	t.log.Warnf("stopped")
	return t.listener.Close()
}

func (t *TcpServer) GracefulStop() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	sig := <-sigCh
	t.log.Warnf("stop by signal: %s", sig)
	return t.Stop()
}
