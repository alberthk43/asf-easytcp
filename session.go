package easytcp

import (
	"fmt"
	"github.com/DarthPestilane/easytcp/message"
	"github.com/google/uuid"
	"net"
	"sync"
	"time"
)

// Session represents a TCP session.
type Session struct {
	id        string              // session's ID. it's a UUID
	conn      net.Conn            // tcp connection
	closeOnce sync.Once           // to make sure we can only close each session one time
	closed    chan struct{}       // to close()
	reqQueue  chan *message.Entry // request queue channel, pushed in readLoop() and popped in router.Router
	respQueue chan *message.Entry // response queue channel, pushed in SendResp() and popped in writeLoop()
	packer    Packer              // to pack and unpack message
	codec     Codec               // encode/decode message data
}

// SessionOption is the extra options for Session.
type SessionOption struct {
	Packer          Packer
	Codec           Codec
	ReadBufferSize  int
	WriteBufferSize int
}

// newSession creates a new Session.
// Parameter conn is the TCP connection,
// opt includes packer, codec, and channel size.
// Returns a Session pointer.
func newSession(conn net.Conn, opt *SessionOption) *Session {
	id := uuid.NewString()
	return &Session{
		id:        id,
		conn:      conn,
		closed:    make(chan struct{}),
		reqQueue:  make(chan *message.Entry, opt.ReadBufferSize),
		respQueue: make(chan *message.Entry, opt.WriteBufferSize),
		packer:    opt.Packer,
		codec:     opt.Codec,
	}
}

// ID returns the session's ID.
func (s *Session) ID() string {
	return s.id
}

// SendResp pushes response message entry to respQueue.
// Returns error if session is closed.
func (s *Session) SendResp(respMsg *message.Entry) error {
	select {
	case <-s.closed:
		// This method is the producer of respQueue,
		// so we close respQueue only in here.
		close(s.respQueue)
		return fmt.Errorf("sessions is closed")
	default:
	}

	s.respQueue <- respMsg
	return nil
}

// Close closes the session by closing all the channels.
func (s *Session) Close() {
	s.closeOnce.Do(func() { close(s.closed) })
}

// readLoop reads TCP connection, unpacks packet payload
// to a MessageEntry, and push to reqQueue channel.
// The above operations are in a loop.
// Parameter readTimeout specified the connection reading timeout.
// The loop will break if any error occurred, or the session is closed.
// After loop ended, this session will be closed.
func (s *Session) readLoop(readTimeout time.Duration) {
	for {
		if readTimeout > 0 {
			if err := s.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
				Log.Errorf("session set read deadline err: %s", err)
				break
			}
		}
		entry, err := s.packer.Unpack(s.conn)
		if err != nil {
			Log.Errorf("session unpack incoming message err: %s", err)
			break
		}
		if entry == nil {
			continue
		}

		select {
		case <-s.closed:
			// This method is the producer of reqQueue,
			// so we close reqQueue only in here.
			close(s.reqQueue)

			Log.Tracef("session read loop exit because session is closed")
			return
		case s.reqQueue <- entry:
		}
	}
	Log.Tracef("session read loop exit because of error")
	s.Close()
}

// writeLoop fetches message from respQueue channel and writes to TCP connection in a loop.
// Parameter writeTimeout specified the connection writing timeout.
// The loop will break if errors occurred, or the session is closed.
// After loop ended, this session will be closed.
func (s *Session) writeLoop(writeTimeout time.Duration) {
FOR:
	for {
		select {
		case <-s.closed:
			Log.Tracef("session write loop exit because session is closed")
			return
		case respMsg, ok := <-s.respQueue:
			if !ok {
				Log.Tracef("session write loop exit because session is closed")
				return
			}
			// pack message
			ackMsg, err := s.packer.Pack(respMsg)
			if err != nil {
				Log.Errorf("session pack response message err: %s", err)
				continue
			}
			if ackMsg == nil {
				continue
			}
			if writeTimeout > 0 {
				if err := s.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
					Log.Errorf("session set write deadline err: %s", err)
					break FOR
				}
			}
			if _, err := s.conn.Write(ackMsg); err != nil {
				Log.Errorf("session conn write err: %s", err)
				break FOR
			}
		}
	}
	s.Close()
	Log.Tracef("session write loop exit because of error")
}
