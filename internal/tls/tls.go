package tls

import (
	"google.golang.org/grpc/metadata"
	"net/http"
	"regexp"
	"strings"
)

const (
	forwardedHeader = "Forwarded"
	httpsProto      = "proto=https"
)

var httpsProtoPattern = regexp.MustCompile(httpsProto)

func AddHTTPSMarker(header http.Header) {
	header.Add(forwardedHeader, httpsProto)
}

func RemoveHTTPSMarker(md metadata.MD) {
	// TODO: what if the request was genuinely forwarded before reaching grpc-dump?
	// in that case we'll be deleting information here that wasn't added by grpc-dump
	delete(md, strings.ToLower(forwardedHeader))
}

func IsTLSRPC(md metadata.MD) bool {
	values := md.Get(forwardedHeader)
	for _, value := range values {
		if value == httpsProto {
			return true
		}
	}

	return false
}

func IsTLSRequest(header http.Header) bool {
	value := header.Get(forwardedHeader)
	return httpsProtoPattern.MatchString(value)
}
