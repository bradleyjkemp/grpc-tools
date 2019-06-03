# grpc-fixture

`grpc-fixture` intercepts all client messages and replays previously recorded server responses.

This is great for:
* Easily running tests using your real client code but with mocked server responses.
* Reproducing client bugs deterministically and without having to make actual requests to servers. 
* Giving demos without internet connection and that you know will work the same every time.

## Command line usage

```
Usage of grpc-fixture:
  -cert string
    	Certificate file to use for serving using TLS.
  -dump string
    	gRPC dump to serve requests from.
  -key string
    	Key file to use for serving using TLS.
  -port int
    	Port to listen on.
```

## Troubleshooting

For troubleshooting see the generic `grpc-proxy` troubleshooting steps [here](../grpc-proxy/README.md).
