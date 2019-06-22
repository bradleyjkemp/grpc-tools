# grpc-proxy

`grpc-proxy` is a minimal configuration library that acts as a local HTTP proxy and transparently proxies all gRPC requests to the requested destination. A gRPC [StreamClientInterceptor](https://godoc.org/google.golang.org/grpc#StreamClientInterceptor) can be registered which will be called for all requests and lets you do anything that gRPC middleware can do.

For example you can build applications that:
* Dump request metadata (e.g. [`grpc-dump`](../grpc-dump/README.md)).
* Respond to requests using saved/mocked responses (e.g. [`grpc-fixture`](../grpc-fixture/README.md)).
* Modify request contents before it is sent to the server and modify the server response before it is returned to the client.

## Basic example

`grpc-proxy` is super simple to use, this short snippet is a simplified version of `grpc-dump` that logs the service names of all intercepted methods:

```go
package main

import (
	"flag"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/grpc-proxy"
	"google.golang.org/grpc"
)

func main() {
	grpc_proxy.RegisterDefaultFlags()
	flag.Parse()
	proxy, _ := grpc_proxy.New(
        grpc_proxy.WithInterceptor(intercept),
        grpc_proxy.DefaultFlags(),
    )
    proxy.Start()
}

func intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	fmt.Println(info.FullMethod)
	return handler(srv, ss)
}
```

## Features

* Acts as a HTTP proxy silently intercepting traffic from all applications that support HTTP proxies.
* Supports both gRPC and gRPC-Web and both Streaming and Unary RPCs.
* Serves TLS and non-TLS traffic on a single port.
* Gracefully falls back to proxying the raw request if it cannot be silently intercepted (e.g. it isn't being run with a valid TLS certificate for the domain)
* Fallback mode for applications that do not support HTTP proxies: applications can be pointed at the proxy directly and an explicit destination specified that all requests will be forwarded to.

## Troubleshooting

### Application requests aren't being intercepted

`grpc-proxy` can only intercept requests if the application supports HTTP proxies. The standard gRPC libraries support HTTP proxies by default but applications can override this behaviour by using a custom Dialer.

Troubleshooting steps:
1. Check whether the application supports HTTP proxies, if your application does not support HTTP proxies, go straight to the instructions for using `grpc-proxy` in [fallback mode](#using-grpc-proxy-in-fallback-mode).
1. Check your application is configured to use the HTTP proxy. This is normally done by setting the `http_proxy` or `all_proxy` environment variable to the address `grpc-proxy` is listening on (e.g. `http_proxy=http://localhost:12345`).

    Some applications interpret these environment variables differently, you may have to set both `http_proxy` and `https_proxy` to point to `grpc-proxy`.
1. Try using `grpc-proxy` as your system proxy. This will mean traffic from all applications will go through the proxy. You can find instructions for this [here](https://software.intel.com/en-us/articles/how-to-set-system-proxy).

    **Warning**: this may cause other applications to not work properly if `grpc-proxy` is unable to proxy its requests properly. It's preferable to only configure the target application to use the proxy to minimise potential disruption.
1. Try using `grpc-proxy` in fallback mode instead (see below for details).

### Using `grpc-proxy` in fallback mode

For applications that do not work with HTTP proxies, `grpc-proxy` can act as an explicit proxy server.

Rather than letting `grpc-proxy` try to transparently intercept requests you should configure your application to connect directly to your proxy and use the Destination setting to tell `grpc-proxy` to send all traffic to the specified host.
