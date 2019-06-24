# Changelog

## v0.1.3 (unreleased)
* Added proxy loop detection so that misconfiguration (e.g. missing/incorrent `--destination` flag) do not cause infinite loops and connection exhaustion.
* `grpc-proxy` now supports requests with gzip compression (however requests are still proxied uncompressed).
* `grpc-dump` now includes a timestamp along with each message sent/received.

## [v0.1.2](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.1.2)
* Fixed bug where the `--destination` flag didn't work (issue [#13](https://github.com/bradleyjkemp/grpc-tools/issues/13)).

## [v0.1.1](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.1.1)
* Added automatic detection of mkcert certificates in the current directory.

## [v0.1.0](https://github.com/bradleyjkemp/grpc-tools/releases/tag/v0.1.0)
* Added grpc-dump, grpc-fixture, grpc-replay.
