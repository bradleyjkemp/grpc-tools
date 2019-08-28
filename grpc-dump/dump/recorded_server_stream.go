package dump

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal/dump_format"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"sync"
	"time"
)

// dumpServerStream wraps a grpc.ServerStream and allows the dump interceptor to record all sent/received messages
type dumpServerStream struct {
	sync.Mutex
	grpc.ServerStream
	messageCounter int

	rpcID      int
	fullMethod string
	output     io.Writer
	logger     logrus.FieldLogger
	decoder    proto_decoder.MessageDecoder
}

func (ss *dumpServerStream) SendMsg(m interface{}) error {
	message := m.([]byte)
	ss.Lock()
	defer ss.Unlock()
	ss.dumpMessage(message, dump_format.ServerMessage)
	return ss.ServerStream.SendMsg(m)
}

func (ss *dumpServerStream) RecvMsg(m interface{}) error {
	err := ss.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	// now m is populated
	message := m.(*[]byte)
	ss.Lock()
	defer ss.Unlock()
	ss.dumpMessage(*message, dump_format.ClientMessage)
	return nil
}

func (ss *dumpServerStream) dumpMessage(message []byte, origin dump_format.MessageOrigin) {
	msgLine := &dump_format.Message{
		Timestamp:     time.Now(),
		Type:          dump_format.MessageLine,
		ID:            ss.rpcID,
		MessageID:     ss.messageCounter,
		MessageOrigin: origin,
		RawMessage:    message,
	}
	ss.messageCounter++

	var err error
	msgLine.Message, err = ss.decoder.Decode(ss.fullMethod, msgLine)
	if err != nil {
		ss.logger.WithError(err).Warn("Failed to decode message")
	}

	line, _ := json.Marshal(msgLine)
	if _, err := fmt.Fprintln(ss.output, string(line)); err != nil {
		ss.logger.WithError(err).Warn("Failed to dump message")
	}
}
