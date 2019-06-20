package marker

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
)

var viaFormat = "HTTP/2.0 %s"

func AddLoopCheck(md metadata.MD, proxyID string) error {
	viaValue := fmt.Sprintf(viaFormat, proxyID)

	via := md.Get("Via")
	if len(via) == 0 {
		md.Set("Via", viaValue)
		return nil
	}

	// via is comma separated
	parts := strings.Split(via[0], ",")
	for _, part := range parts {
		if part == viaValue {
			return status.Error(codes.Internal, fmt.Sprintf("proxy loop detected, request already handled by %s", proxyID))
		}
	}

	md.Set("Via", fmt.Sprintf("%s, %s", via, viaValue))
	return nil
}
