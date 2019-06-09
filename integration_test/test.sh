#!/usr/bin/env bash
killall grpc-fixture
killall grpc-dump
set -e

testDir=$(dirname "${BASH_SOURCE}")
pushd "${testDir}"
certFile="_wildcard.github.io.pem"
keyFile="_wildcard.github.io-key.pem"

if [[ ! -f "$certFile" ]]; then
    echo "required file $certFile doesn't exist"
    exit 1
fi
if [[ ! -f "$keyFile" ]]; then
    echo "required file $keyFile doesn't exist"
    exit 1
fi

go build github.com/bradleyjkemp/grpc-tools/grpc-fixture
go build github.com/bradleyjkemp/grpc-tools/grpc-dump
go build github.com/bradleyjkemp/grpc-tools/grpc-replay

# grpc-fixture serves mock RPCs
./grpc-fixture \
    --dump=test-dump.json \
    --port=16353 \
    --cert="${certFile}" \
    --key="${keyFile}" &
fixturePID=$!

# grpc-dump will dump the requests and responses to the fixture
HTTP_PROXY=localhost:16353 ./grpc-dump \
    --port=16354 \
    --cert="${certFile}" \
    --key="${keyFile}" > test-result.json &
dumpPID=$!

sleep 1 # wait for servers to start up

# grpc-replay makes request which are logged by grpc-dump and responded to by grpc-fixture
HTTP_PROXY=localhost:16354 ./grpc-replay \
    --dump=test-dump.json

kill ${fixturePID}
kill ${dumpPID}

wait

# Now check that the two results match
cmp test-result.json test-golden.json || (echo "Results are different"; exit 1)
echo "Test passes"

# Clean up
rm ./grpc-fixture
rm ./grpc-dump
rm ./grpc-replay
rm test-result.json
popd
