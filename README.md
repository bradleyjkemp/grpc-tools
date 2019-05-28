# grpc-tools

A suite of tools for gRPC debugging and development.

![demo](demo.svg "Simple grpc-dump demo")

Tools include:
* [`grpc-dump`](grpc-dump): a small gRPC proxy that dumps RPC details for debugging, later analysis/replay.
* `grpc-replay`: takes the output from `grpc-dump` and replays requests to the server.
* `grpc-fixture`: a proxy that takes the output from `grpc-dump` and replays responses to requests.
* `grpc-proxy`: the library that the above tools are built on top of.

These tools are in alpha so expect breaking changes between releases. See the [changelog](CHANGELOG.md) for full details.

## grpc-dump ([more details](grpc-dump/README.md))

Basic usage:
```bash
# in one terminal, start the proxy
grpc-dump --port=12345

# in another, run your application pointing it at the proxy
HTTP_PROXY=localhost:12345 my-app

# all the requests being made will be printed out in the first terminal
```
