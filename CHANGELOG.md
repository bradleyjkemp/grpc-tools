# Changelog

## [v0.2.5](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.2.5)
* Added grpc-proxy options `WithServerOptions` and `WithDialOptions`, deprecated `WithOptions` [#71](https://github.com/bradleyjkemp/grpc-tools/pull/71).
* Added a new command-line option, `--interface` to allow choosing which network interface `grpc-proxy` listens on [#81](https://github.com/bradleyjkemp/grpc-tools/pull/81).

## [v0.2.4](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.2.4)
* Added a new command-line option, `--tls_secrets_file`, to write the [TLS master secrets file](https://developer.mozilla.org/en-US/docs/Mozilla/Projects/NSS/Key_Log_Format) to the specified path. This can be used later with different application, such as Wireshark, to decrypt the gRPC traffic  [#63](https://github.com/bradleyjkemp/grpc-tools/pull/63).
* (Issue [#43](https://github.com/bradleyjkemp/grpc-tools/issues/43)) Dumping response header and trailer metadata [#63](https://github.com/bradleyjkemp/grpc-tools/pull/63).
* Fixed bug where HTTP/2 traffic was not intercepted correctly  [#63](https://github.com/bradleyjkemp/grpc-tools/pull/63).
* Some linter recommendations [#63](https://github.com/bradleyjkemp/grpc-tools/pull/63).

## [v0.2.3](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.2.3)
* Added a defer-recover handler so that any panics in the `grpc-proxy` interceptor do not kill the proxy [#53](https://github.com/bradleyjkemp/grpc-tools/pull/53).
* Added a new command-line option, `--system_proxy`, to automatically enable grpc-dump/grpc-fixture as the system proxy on MacOS [#55](https://github.com/bradleyjkemp/grpc-tools/pull/55).

## [v0.2.2](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.2.2)
* Improved behaviour and logging when a message fails to be decoded [#51](https://github.com/bradleyjkemp/grpc-tools/pull/51).

## [v0.2.1](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.2.1)
* Fixed bug where `grpc-dump` would not forward errors from server to client [#45](https://github.com/bradleyjkemp/grpc-tools/pull/45).
* `grpc-proxy` now will transparently forward all non-HTTP traffic to the original destination [#28](https://github.com/bradleyjkemp/grpc-tools/pull/28).
* When the `--proto_roots` or `--proto_descriptors` flags are used, `grpc-replay` and `grpc-fixture` will marshall messages from the human readable form instead of using the raw message.

## [v0.2.0](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.2.0)
* Added proxy loop detection so that misconfiguration (e.g. missing/incorrent `--destination` flag) do not cause infinite loops and connection exhaustion.
* `grpc-proxy` now supports requests with gzip compression (however requests are still proxied uncompressed).
* `grpc-dump` now includes a timestamp along with each message sent/received.
* **Breaking Change**: `grpc-dump` now reports RPC error codes as their human-readable string form instead of a raw integer code.

## [v0.1.2](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.1.2)
* Fixed bug where the `--destination` flag didn't work (issue [#13](https://github.com/bradleyjkemp/grpc-tools/issues/13)).

## [v0.1.1](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.1.1)
* Added automatic detection of mkcert certificates in the current directory.

## [v0.1.0](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.1.0)
* Added grpc-dump, grpc-fixture, grpc-replay.
