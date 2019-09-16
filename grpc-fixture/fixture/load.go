package fixture

import (
	"encoding/json"
	"github.com/bradleyjkemp/grpc-tools/internal/dump_format"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"github.com/pkg/errors"
	"io"
	"os"
)

// map of service name to message tree
type fixture map[string]*messageTree

type messageTree struct {
	origin       dump_format.MessageOrigin
	raw          string
	nextMessages []*messageTree
}

type rpcInfo struct {
	*dump_format.RPC
	*messageTree
}

// load fixture creates a Trie-like structure of messages
func loadFixture(dumpPath string, encoder proto_decoder.MessageEncoder) (fixture, error) {
	dumpFile, err := os.Open(dumpPath)
	if err != nil {
		return nil, err
	}

	dumpDecoder := json.NewDecoder(dumpFile)
	fixture := map[string]*messageTree{}
	rpcs := map[int64]*rpcInfo{}

	for {
		line := &dump_format.DecodedLine{}
		err := dumpDecoder.Decode(line)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch l := line.Get().(type) {
		case *dump_format.RPC:
			// First time we've seen an RPC so initialise to point at the top of the message tree
			if fixture[l.StreamName()] == nil {
				fixture[l.StreamName()] = &messageTree{}
			}
			rpcs[l.ID] = &rpcInfo{l, fixture[l.StreamName()]}

		case *dump_format.Message:
			rpc := rpcs[l.ID]
			messageTreeNode := rpc.messageTree
			msgBytes, err := encoder.Encode(rpc.StreamName(), l)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get message bytes")
			}
			var foundExisting *messageTree
			for _, nextMessage := range messageTreeNode.nextMessages {
				if nextMessage.origin == l.MessageOrigin && nextMessage.raw == string(msgBytes) {
					foundExisting = nextMessage
					break
				}
			}
			if foundExisting == nil {
				foundExisting = &messageTree{
					origin:       l.MessageOrigin,
					raw:          string(msgBytes),
					nextMessages: nil,
				}
				messageTreeNode.nextMessages = append(messageTreeNode.nextMessages, foundExisting)
			}
			rpc.messageTree = foundExisting

		case *dump_format.Status:
			// TODO: handle RPC statuses (issue #24)

		default:
			return nil, errors.Errorf("unknown line type %T", line)
		}
	}

	return fixture, nil
}
