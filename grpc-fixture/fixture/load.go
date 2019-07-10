package fixture

import (
	"encoding/json"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"io"
	"os"
)

// map of service name to message tree
type fixture map[string]*messageTree

type messageTree struct {
	origin       internal.MessageOrigin
	raw          string
	nextMessages []*messageTree
}

// load fixture creates a Trie-like structure of messages
func loadFixture(dumpPath string, encoder proto_decoder.MessageEncoder) (fixture, error) {
	dumpFile, err := os.Open(dumpPath)
	if err != nil {
		return nil, err
	}

	dumpDecoder := json.NewDecoder(dumpFile)
	fixture := map[string]*messageTree{}

	for {
		rpc := internal.RPC{}
		err := dumpDecoder.Decode(&rpc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if fixture[rpc.StreamName()] == nil {
			fixture[rpc.StreamName()] = &messageTree{}
		}
		messageTreeNode := fixture[rpc.StreamName()]
		for _, msg := range rpc.Messages {
			msgBytes, err := encoder.Encode(rpc.StreamName(), msg)
			if err != nil {
				return nil, err
			}
			var foundExisting *messageTree
			for _, nextMessage := range messageTreeNode.nextMessages {
				if nextMessage.origin == msg.MessageOrigin && nextMessage.raw == string(msgBytes) {
					foundExisting = nextMessage
					break
				}
			}
			if foundExisting == nil {
				foundExisting = &messageTree{
					origin:       msg.MessageOrigin,
					raw:          string(msgBytes),
					nextMessages: nil,
				}
				messageTreeNode.nextMessages = append(messageTreeNode.nextMessages, foundExisting)
			}

			messageTreeNode = foundExisting
		}
	}

	return fixture, nil
}
