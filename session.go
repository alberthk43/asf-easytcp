package easytcp

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a TCP session.
type Session interface {
	// ID returns current session's id.
	ID() interface{}

	// SetID sets current session's id.
	SetID(id interface{})

	// Send sends the ctx to the respStream.
	Send(ctx Context) bool

	// Codec returns the codec, can be nil.
	Codec() Codec

	// Close closes current session.
	Close()

	// AllocateContext gets a Context ships with current session.
	AllocateContext() Context

	// Conn returns the underlined connection.
	Conn() net.Conn

	// AfterCreateHook blocks until session's on-create hook triggered.
	AfterCreateHook() <-chan struct{}

	// AfterCloseHook blocks until session's on-close hook triggered.
	AfterCloseHook() <-chan struct{}
}

// 对于连接的封装, 包含了实际的conn以及一些额外的信息
type session struct {
	id               interface{}   // session's ID.
	conn             net.Conn      // tcp connection
	closedC          chan struct{} // to close when read/write loop stopped
	closeOnce        sync.Once     // ensure one session only close once
	afterCreateHookC chan struct{} // to close after session's on-create hook triggered
	afterCloseHookC  chan struct{} // to close after session's on-close hook triggered
	respStream       chan Context  // response queue channel, pushed in Send() and popped in writeOutbound()
	packer           Packer        // to pack and unpack message
	codec            Codec         // encode/decode message data
	ctxPool          sync.Pool     // router context pool
	asyncRouter      bool          // calls router HandlerFunc in a goroutine if false
}

// sessionOption is the extra options for session.
type sessionOption struct {
	Packer        Packer
	Codec         Codec
	respQueueSize int
	asyncRouter   bool
}

// newSession creates a new session.
// Parameter conn is the TCP connection,
// opt includes packer, codec, and channel size.
// Returns a session pointer.
func newSession(conn net.Conn, opt *sessionOption) *session {
	return &session{
		id:               uuid.NewString(), // use uuid as default
		conn:             conn,
		closedC:          make(chan struct{}),
		afterCreateHookC: make(chan struct{}),
		afterCloseHookC:  make(chan struct{}),
		respStream:       make(chan Context, opt.respQueueSize),
		packer:           opt.Packer,
		codec:            opt.Codec,
		ctxPool:          sync.Pool{New: func() interface{} { return newContext() }},
		asyncRouter:      opt.asyncRouter,
	}
}

// ID returns the session's id.
func (s *session) ID() interface{} {
	return s.id
}

// SetID sets session id.
// Can be called in server.OnSessionCreate() callback.
func (s *session) SetID(id interface{}) {
	s.id = id
}

// Send pushes response message to respStream.
// Returns false if session is closed or ctx is done.
func (s *session) Send(ctx Context) (ok bool) {
	select {
	case <-ctx.Done(): // 要正确处理ctx的关闭的case
		return false
	case <-s.closedC:
		return false
	case s.respStream <- ctx: // 数据返回给客户端
		return true
	}
}

// Codec implements Session Codec.
// 相当于业务没有实际的调用的方式, 仅仅是"存储了"一个Codec, 可以在需要的时候给出来
func (s *session) Codec() Codec {
	return s.codec
}

// Close closes the session, but doesn't close the connection.
// The connection will be closed in the server once the session's closed.
func (s *session) Close() {
	s.closeOnce.Do(func() { close(s.closedC) }) // 用于确保只close一次, 其实只是关闭了一个channel, 给出了关闭的信号, 不是实际close conn
}

// AfterCreateHook blocks until session's on-create hook triggered.
func (s *session) AfterCreateHook() <-chan struct{} {
	return s.afterCreateHookC
}

// AfterCloseHook blocks until session's on-close hook triggered.
func (s *session) AfterCloseHook() <-chan struct{} {
	return s.afterCloseHookC
}

// AllocateContext gets a Context from pool and reset all but session.
func (s *session) AllocateContext() Context {
	c := s.ctxPool.Get().(*routeContext) // 使用了对象池创建了一个Context的实例
	c.reset()                            // 防止对象池拿出来的Context有脏数据
	c.SetSession(s)                      // 设置session到Context中
	return c
}

// Conn returns the underlined connection instance.
func (s *session) Conn() net.Conn {
	return s.conn
}

// readInbound reads message packet from connection in a loop.
// And send unpacked message to reqQueue, which will be consumed in router.
// The loop breaks if errors occurred or the session is closed.
// 是一个协程, 用于读取连接中的数据
// Router表示了协议的处理逻辑, 是应用层的逻辑
func (s *session) readInbound(router *Router, timeout time.Duration) {
	for {
		// 如果session关闭了, 则退出, default case是为了避免阻塞
		select {
		case <-s.closedC: // 当read或者write的loop结束, 就会设置这个信号, 下次另外的一个循环就会退出
			return
		default:
		}

		// 如果有超时, 则设置超时时间
		if timeout > 0 {
			if err := s.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
				_log.Errorf("session %s set read deadline err: %s", s.id, err)
				break
			}
		}

		// s.pack是一个interface, 用于用户自定义的协议, 用于解析数据包
		// 这里的Unpack是否可以设计为异步的? 因为可能unpack会有较高的CPU消耗, 不要卡主网络读取是否更好?
		reqMsg, err := s.packer.Unpack(s.conn)
		if err != nil {
			logMsg := fmt.Sprintf("session %s unpack inbound packet err: %s", s.id, err)
			if err == io.EOF { // 因为是直接从conn读取数据, 所以EOF表示连接关闭, 是正常的情况
				_log.Tracef(logMsg)
			} else {
				_log.Errorf(logMsg)
			}
			break // 如果解析失败, 则退出, 会关闭session, 在函数的最后进行了 s.Close() 操作
		}
		if reqMsg == nil { // 容错的处理
			continue
		}

		// 处理数据包, 如果是异步的, 则开启一个协程处理, 否则直接处理
		if s.asyncRouter {
			go s.handleReq(router, reqMsg)
		} else {
			s.handleReq(router, reqMsg)
		}
	}
	_log.Tracef("session %s readInbound exit because of error", s.id)
	s.Close()
}

func (s *session) handleReq(router *Router, reqMsg *Message) {
	ctx := s.AllocateContext().SetRequestMessage(reqMsg) // 上面会默认设置 session自身, 这里又设置了reqMsg
	router.handleRequest(ctx)
	s.Send(ctx) // ctx包含了处理的结果
}

// writeOutbound fetches message from respStream channel and writes to TCP connection in a loop.
// Parameter writeTimeout specified the connection writing timeout.
// The loop breaks if errors occurred, or the session is closed.
func (s *session) writeOutbound(writeTimeout time.Duration) {
	for {
		var ctx Context
		select {
		case <-s.closedC: // close关闭了这个chan, 下次另外一个read/write的loop就会关闭退出
			return
		case ctx = <-s.respStream: // 需要返回给客户端的ctx, 继续执行下面的发送逻辑
		}

		outboundBytes, err := s.packResponse(ctx)
		if err != nil {
			_log.Errorf("session %s pack outbound message err: %s", s.id, err)
			continue
		}
		if outboundBytes == nil {
			continue
		}

		if writeTimeout > 0 { // 每次发送数据前, 都设置超时时间, 注意这个用法不是初始化设置一次就行的
			if err := s.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				_log.Errorf("session %s set write deadline err: %s", s.id, err)
				break
			}
		}

		if _, err := s.conn.Write(outboundBytes); err != nil {
			_log.Errorf("session %s conn write err: %s", s.id, err)
			break
		}
	}
	s.Close()
	_log.Tracef("session %s writeOutbound exit because of error", s.id)
}

func (s *session) packResponse(ctx Context) ([]byte, error) {
	defer s.ctxPool.Put(ctx) // ctx的生命周期在这里结束, 放回对象池
	if ctx.Response() == nil {
		return nil, nil
	}
	return s.packer.Pack(ctx.Response())
}
