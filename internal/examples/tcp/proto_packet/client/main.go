package main

import (
	"net"
	"time"

	"github.com/DarthPestilane/easytcp"
	"github.com/DarthPestilane/easytcp/internal/examples/fixture"
	"github.com/DarthPestilane/easytcp/internal/examples/tcp/proto_packet/common"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.SetLevel(logrus.DebugLevel)
}

func main() {
	conn, err := net.Dial("tcp", fixture.ServerAddr)
	if err != nil {
		panic(err)
	}

	packer := &common.CustomPacker{}
	codec := &easytcp.ProtobufCodec{}

	go func() {
		for {
			var id = common.ID_FooReqID
			req := &common.FooReq{
				Bar: "bar",
				Buz: 22,
			}

			// codec 用于对于proto进行编码, 但是没有tcp真正的消息封装的概念
			// packer 用于封装tcp的消息, 但是不关心打包的 bytes 是什么
			// codec 更贴近业务层

			data, err := codec.Encode(req)
			if err != nil {
				panic(err)
			}
			packedMsg, err := packer.Pack(easytcp.NewMessage(id, data))
			if err != nil {
				panic(err)
			}
			if _, err := conn.Write(packedMsg); err != nil {
				panic(err)
			}
			log.Debugf("send | id: %d; size: %d; data: %s", id, len(data), req.String())
			time.Sleep(time.Second)
		}
	}()

	for {
		msg, err := packer.Unpack(conn)
		if err != nil {
			panic(err)
		}
		var respData common.FooResp
		if err := codec.Decode(msg.Data(), &respData); err != nil {
			panic(err)
		}
		log.Infof("recv | id: %d; size: %d; data: %s", msg.ID(), len(msg.Data()), respData.String())
	}
}
