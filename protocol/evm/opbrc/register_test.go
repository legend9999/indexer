package opbrc

import (
	"math/big"
	"testing"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/devents"
)

func TestProtocol_processRegister(t *testing.T) {
	registerMd := devents.MetaData{
		Chain:    chainName,
		Protocol: protocolName,
		Operate:  OperateRegister,
		Tick:     "legend3",
		Data:     registerJsonStr,
	}

	protocol := intEnv()
	tx := new(xycommon.RpcTransaction)
	tx.From = "0x63EeFb268BD21C57B5eEbA1cbC909695BFA66f4E"
	tx.BlockNumber = big.NewInt(17652371)
	register, err := protocol.checkRegister(nil, tx, &registerMd)

	if err != nil {
		t.Fatalf("checkRegister failed: %v", err)
		return
	}
	t.Log(register)
}
