# grpc-dump

`grpc-dump` intercepts all gRPC requests and responses and logs all the request metadata in a JSON stream.

These streams can be used for all sorts of applications including:
* Debugging what requests are being made by an application and to which servers.
* Using tools like [`jq`](https://stedolan.github.io/jq/) to easily filter and transform the data for analysis.
* Using [`grpc-fixture`](../grpc-fixture/README.md) to intercept future gRPC requests from the application and replay the responses saved in the dump.
* Using [`grpc-replay`](../grpc-replay/README.md) to replay the requests exactly as the client made them and check that the server still responds the same way.

## Command line interface
```
Usage of grpc-dump:
  -cert string
    	Certificate file to use for serving using TLS.
  -destination string
    	Destination server to forward requests to if no destination can be inferred from the request itself. This is generally only used for clients not supporting HTTP proxies.
  -key string
    	Key file to use for serving using TLS.
  -port int
    	Port to listen on.
  -proto_descriptors string
    	A comma separated list of proto descriptors to load gRPC service definitions from.
  -proto_roots string
    	A comma separated list of directories to search for gRPC service definitions.
```

## Troubleshooting

For troubleshooting see the generic `grpc-proxy` troubleshooting steps [here](../grpc-proxy/README.md).
