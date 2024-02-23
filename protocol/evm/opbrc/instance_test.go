package opbrc

import (
	"math/big"
	"net/http"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/config"
	"github.com/uxuycom/indexer/dcache"
	"github.com/uxuycom/indexer/devents"
	"github.com/uxuycom/indexer/storage"
	"github.com/uxuycom/indexer/xylog"
)

const (
	deployJsonStr   = `{"p":"opbrc","op":"deploy","tick":"legend1","max":"10000","mspan":"1","sm":"5","mcount":"100","cost":"10000000000000"}`
	registerJsonStr = `{"p":"opbrc","op":"register","tick":"legend1"}`
)

func intEnv() *Protocol {
	var (
		cfg        config.Config
		flagConfig string
	)

	// init
	runtime.GOMAXPROCS(runtime.NumCPU())

	// load configs
	config.LoadConfig(&cfg, flagConfig)

	// enable profile
	if cfg.Profile != nil && cfg.Profile.Enabled {
		go func() {
			listen := cfg.Profile.Listen
			if listen == "" {
				listen = ":6060"
			}
			if err := http.ListenAndServe(listen, nil); err != nil {
				xylog.Logger.Infof("start profile err:%v", err)
			}
		}()
	}

	// set log debug level
	if lv, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
		xylog.InitLog(lv, cfg.LogPath)
	}

	dbClient, err := storage.NewDbClient(&cfg.Database)
	if err != nil {
		xylog.Logger.Fatalf("db init err:%v", err)
	}

	dCache := dcache.NewManager(dbClient, cfg.Chain.ChainName)
	return NewProtocol(dCache)

}
func TestProtocol_UpdateInscriptionExtOnDeploy(t *testing.T) {
	deployMd := devents.MetaData{
		Chain:    chainName,
		Protocol: protocolName,
		Operate:  "deploy",
		Tick:     "legend1",
		Data:     deployJsonStr,
	}
	protocol := intEnv()

	tx := new(xycommon.RpcTransaction)
	tx.From = "0x63EeFb268BD21C57B5eEbA1cbC909695BFA66f4E"
	tx.BlockNumber = big.NewInt(17653478)
	block := new(xycommon.RpcBlock)
	block.Number = big.NewInt(17653478)
	deploy, err := protocol.checkDeploy(block, tx, &deployMd)
	if err != nil {
		t.Fatalf("err:%v", err)
		return
	}
	t.Log(deploy)
}
