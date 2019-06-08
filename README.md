# grpc-tools

A suite of tools for gRPC debugging and development.

![demo](demo.svg "Simple grpc-dump demo")

Tools include:
* [`grpc-dump`](#grpc-dump): a small gRPC proxy that dumps RPC details for debugging, later analysis/replay.
* [`grpc-replay`](grpc-replay): takes the output from `grpc-dump` and replays requests to the server.
* [`grpc-fixture`](#grpc-fixture): a proxy that takes the output from `grpc-dump` and replays saved responses to client requests.
* [`grpc-proxy`](grpc-proxy): the library that the above tools are built on top of.

These tools are in alpha so expect breaking changes between releases. See the [changelog](CHANGELOG.md) for full details.

Installation:
```bash
go install github.com/bradleyjkemp/grpc-tools/...
```

## grpc-dump

Basic usage:
```bash
# start the proxy (leave out the --port flag to automatically pick on)
grpc-dump --port=12345

# in another terminal, run your application pointing it at the proxy
HTTP_PROXY=localhost:12345 my-app

# all the requests made by the application will be logged to standard output in the grpc-dump window e.g.
# {"service": "echo", "method": "Hi", "messages": ["....."] }
```

Many applications expect to talk to a gRPC server over TLS. For this you need to use the `--key` and `--cert` flags to point `grpc-dump` to certificates valid for the domains your application connects to. The recommended way to generate these files is via the excellent [`mkcert`](https://github.com/FiloSottile/mkcert) tool:
```bash
# Configure your system to trust mkcert certificates
mkcert -install

# Generate certificates for domains you want to intercept connections to
mkcert mydomain.com *.mydomain.com

# Start grpc-dump using the key and certificate created by mkcert
grpc-dump --key=mydomain.com-key.pem --cert=mydomain.com.pem
```

More details for using `grpc-dump` can be found [here](grpc-dump/README.md).

## grpc-fixture

Basic usage:
```bash
# save the (stdout) output of grpc-dump to a file
grpc-dump --port=12345 > myapp.dump

# in another, run your application pointing it at the proxy
HTTP_PROXY=localhost:12345 my-app

# all the requests made by the application will be logged in the grpc-dump window
```

For applications that expect a TLS server, the same `--key` and `--cert` flags can be used as described above for `grpc-dump`.

More details for using `grpc-fixture` can be found [here](grpc-fixture/README.md).
