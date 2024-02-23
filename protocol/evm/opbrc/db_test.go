package opbrc

import (
	"math/rand"
	"testing"
)

func TestProtocol_mintFinish(t *testing.T) {
	protocol := intEnv()
	db := protocol.cache.GetDBClient().SqlDB
	protocol.updateProgressMintFinish(db, "legend1")
}

func TestProtocol_updateSettledBlockNumber(t *testing.T) {
	protocol := intEnv()
	db := protocol.cache.GetDBClient().SqlDB
	protocol.updateSettledBlockNumber(db, "legend1", 100)
}

func TestProtocol_mintQty(t *testing.T) {
	protocol := intEnv()
	db := protocol.cache.GetDBClient().SqlDB
	mintQty, err := protocol.mintedQty(db, "legend1")
	if err != nil {
		t.Fatalf("%s", err)
		return
	}
	t.Logf("%s minted qty = %s", "legend1", mintQty)
}

func TestProtocol_UpdateMintTimes(t *testing.T) {
	protocol := intEnv()
	mintTimes := map[string]uint64{
		"aaa": uint64(rand.Int()),
		"bbb": uint64(rand.Int()),
	}
	protocol.updateMintTimes("legend1", mintTimes)
}

func TestProtocol_addMintTimes(t *testing.T) {
	protocol := intEnv()
	mintTimes := map[string]uint64{
		"aaa111": 1,
		"bbb111": 2,
	}
	protocol.insertMintTimes("legend1", mintTimes)
}
