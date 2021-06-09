package session

import (
	"bytes"
	"fmt"
	"github.com/DarthPestilane/easytcp/logger"
	"github.com/DarthPestilane/easytcp/packet"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net"
)

// UDPSession represents a UDP session.
// Implements Session interface.
type UDPSession struct {
	id         string               // session's id. a uuid
	conn       *net.UDPConn         // udp connection
	log        *logrus.Entry        // logger
	closed     chan struct{}        // represents whether the session is closed. will be closed in Close() method
	reqQueue   chan *packet.Request // a non-buffer channel, pushed in ReadIncomingMsg(), popped in router.Router
	ackQueue   chan []byte          // a non-buffer channel, pushed in SendResp(), popped in Write()
	msgPacker  packet.Packer        // pack/unpack message packet
	msgCodec   packet.Codec         // encode/decode message data
	remoteAddr *net.UDPAddr         // UDP remote address, used to conn.WriteToUDP(remoteAddr)
}

var _ Session = &UDPSession{}

// NewUDP creates a new UDPSession.
// Parameter conn is the UDP connection, addr will be used as remote UDP peer address to write to,
// packer and codec will be used to pack/unpack and encode/decode message.
// Returns a UDPSession pointer.
func NewUDP(conn *net.UDPConn, addr *net.UDPAddr, packer packet.Packer, codec packet.Codec) *UDPSession {
	id := uuid.NewString()
	return &UDPSession{
		id:         id,
		conn:       conn,
		closed:     make(chan struct{}),
		log:        logger.Default.WithField("sid", id).WithField("scope", "session.UDPSession"),
		reqQueue:   make(chan *packet.Request),
		ackQueue:   make(chan []byte),
		msgPacker:  packer,
		msgCodec:   codec,
		remoteAddr: addr,
	}
}

// ID implements the Session ID method.
// Returns session's ID.
func (s *UDPSession) ID() string {
	return s.id
}

// MsgCodec implements the Session MsgCodec method.
// Returns the message codec bound to session.
func (s *UDPSession) MsgCodec() packet.Codec {
	return s.msgCodec
}

// RecvReq implements the Session RecvReq method.
// Returns reqQueue channel which contains *packet.Request.
func (s *UDPSession) RecvReq() <-chan *packet.Request {
	return s.reqQueue
}

// SendResp implements the Session SendResp method.
// Encode and pack resp and push to ackQueue channel.
// It won't panic even when ackQueue channel is closed.
// It returns error when encode or pack failed.
func (s *UDPSession) SendResp(resp *packet.Response) (closed bool, _ error) {
	data, err := s.msgCodec.Encode(resp.Data)
	if err != nil {
		return false, fmt.Errorf("encode response data err: %s", err)
	}
	msg, err := s.msgPacker.Pack(resp.ID, data)
	if err != nil {
		return false, fmt.Errorf("pack response data err: %s", err)
	}
	return !s.safelyPushAckQueue(msg), nil
}

// ReadIncomingMsg reads and unpacks the incoming message packet inMsg,
// then creates a packet.Request and push to reqQueue.
// Returns error when unpack failed.
func (s *UDPSession) ReadIncomingMsg(inMsg []byte) error {
	msg, err := s.msgPacker.Unpack(bytes.NewReader(inMsg))
	if err != nil {
		s.log.Tracef("unpack incoming message err: %s", err)
		return err
	}
	req := &packet.Request{
		ID:      msg.GetID(),
		RawSize: msg.GetSize(),
		RawData: msg.GetData(),
	}
	s.safelyPushReqQueue(req)
	return nil
}

// Write writes the message to a UDP peer.
// Will stop as soon as <-done or ackQueue closed,
// or when connection failed to write.
func (s *UDPSession) Write(done <-chan struct{}) {
	select {
	case <-done:
		return
	case msg, ok := <-s.ackQueue:
		if !ok {
			return
		}
		if _, err := s.conn.WriteToUDP(msg, s.remoteAddr); err != nil {
			s.log.Tracef("conn write err: %s", err)
			return
		}
	}
}

// Close closes the session by closing all the channels.
// NOT safe in concurrency, each session should call Close() only for one time.
func (s *UDPSession) Close() {
	close(s.closed)
	close(s.reqQueue)
	close(s.ackQueue)
}

func (s *UDPSession) safelyPushReqQueue(req *packet.Request) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Tracef("push reqQueue panics: %+v", r)
		}
	}()
	s.reqQueue <- req
}

func (s *UDPSession) safelyPushAckQueue(msg []byte) (ok bool) {
	ok = true
	defer func() {
		if r := recover(); r != nil {
			ok = false
			s.log.Tracef("push ackQueue panics: %+v", r)
		}
	}()
	s.ackQueue <- msg
	return ok
}