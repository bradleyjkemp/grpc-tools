# grpc-tools

A suite of tools for gRPC debugging

Tools include:
* `grpc-dump`: a small gRPC proxy that dumps RPC details for debugging, later analysis/replay.
* `grpc-replay` (todo): takes the output from `grpc-dump` and replays requests to the server.
* `grpc-mock` (todo): a proxy that takes the output from `grpc-dump` and replays responses to requests.
* `grpc-proxy`: the library that the above tools are built on top of.

These tools are in alpha so expect breaking changes.
