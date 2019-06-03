# grpc-replay

`grpc-replay` takes the output of `grpc-dump` and replays the exact requests to the servers and checks that the responses match.

## Command line usage
```
Usage of grpc-replay:
  -destination string
    	Destination server to forward requests to. By default the destination for each RPC is autodetected from the dump metadata.
  -dump string
    	The gRPC dump to replay requests from
```
