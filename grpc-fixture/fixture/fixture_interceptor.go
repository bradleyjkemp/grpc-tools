package fixture

import (
	"github.com/bradleyjkemp/grpc-tools/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fixtureInterceptor implements a gRPC.StreamingServerInterceptor that replays saved responses
func (f fixture) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, _ grpc.StreamHandler) error {
	messageTreeNode := f[info.FullMethod]

	if messageTreeNode == nil {
		return status.Error(codes.Unavailable, "no saved responses found for method "+info.FullMethod)
	}

	for {
		// possibility that server sends the first method
		serverFirst := len(messageTreeNode.nextMessages) > 0
		for _, message := range messageTreeNode.nextMessages {
			serverFirst = serverFirst && message.origin == internal.ServerMessage
		}

		if serverFirst {
			for _, message := range messageTreeNode.nextMessages {
				if message.origin == internal.ServerMessage {
					err := ss.SendMsg([]byte(message.raw))
					if err != nil {
						return err
					}

					// recurse deeper into the tree
					messageTreeNode = message

					// found the first server message so break
					break
				}
			}
		} else {
			// wait for a client message and then proceed based on its contents
			var receivedMessage []byte
			err := ss.RecvMsg(&receivedMessage)
			if err != nil {
				return err
			}
			var found bool
			for _, message := range messageTreeNode.nextMessages {
				if message.origin == internal.ClientMessage && message.raw == string(receivedMessage) {
					// found the matching message so recurse deeper into the tree
					messageTreeNode = message
					found = true
					break
				}
			}

			if !found {
				return status.Errorf(codes.Unavailable, "no matching saved responses for method %s and message", info.FullMethod)
			}
		}

		if len(messageTreeNode.nextMessages) == 0 {
			// end of the exchange
			return nil
		}
	}
}
