package fixture

import (
	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/bradleyjkemp/grpc-tools/internal/proto_decoder"
	"testing"
)

func TestLoadFixture(t *testing.T) {
	r, err := proto_decoder.NewFileResolver("../../.")
	if err != nil {
		t.Fatal(err)
	}

	fixture, err := loadFixture("../../integration_test/test-fixture.json", proto_decoder.NewEncoder(r))
	if err != nil {
		t.Fatal(err)
	}

	cupaloy.SnapshotT(t, fixture)
}
