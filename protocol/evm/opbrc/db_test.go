package opbrc

import (
	"fmt"
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
func TestProtocol_updateMintTimes2(t *testing.T) {
	protocol := intEnv()
	mintTimes := map[string]uint64{
		"aaa111": 1,
		"bbb111": 2,
	}

	for i := 0; i < 20000; i++ {
		mintTimes[fmt.Sprintf("%d", i)] = 1
	}
	updateMintTimes2, err := protocol.updateMintTimes2("uxuy1", mintTimes)
	if err != nil {
		t.Fatalf("%s", err)
		return
	}
	t.Log(updateMintTimes2)
}
