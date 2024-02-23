// Copyright (c) 2023-2024 The UXUY Developer Team
// License:
// MIT License

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//SOFTWARE

package opbrc

import (
	"regexp"
	"strings"
	"sync"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/dcache"
	"github.com/uxuycom/indexer/devents"
	"github.com/uxuycom/indexer/model"
	"github.com/uxuycom/indexer/xyerrors"
	"github.com/uxuycom/indexer/xylog"
)

const (
	protocolName = "opbrc"
	chainName    = "opbnb"
)

const (
	OperateRegister string = "register"
	OperateProxyPay string = "proxy_pay"
	OperateList     string = "list"
	OperateBuy      string = "buy"
)

type Protocol struct {
	cache                        *dcache.Manager
	tickMap                      *sync.Map // tickName -> *tickEntity
	tickExtMap                   *sync.Map // tickName -> tick ext
	allAddressMintTimesMap       *sync.Map // tickName -> map of{address -> mintTimes}
	allAddressCurrentSmMintTxMap *sync.Map // tickName -> map of{address -> []*Mint{}]}
}

func NewProtocol(cache *dcache.Manager) *Protocol {
	dbClient := cache.GetDBClient()
	if err := dbClient.SqlDB.AutoMigrate(&model.OpbrcAddressMintTimes{}, &model.OpbrcInscriptionExt{}, &model.OpbrcMarketPlaceTx{}); err != nil {
		xylog.Logger.Fatalf("new opbrc protocal fatal %v", err)
		return nil
	}

	result := &Protocol{
		cache:                        cache,
		tickMap:                      &sync.Map{},
		tickExtMap:                   &sync.Map{},
		allAddressMintTimesMap:       &sync.Map{},
		allAddressCurrentSmMintTxMap: &sync.Map{},
	}

	inscriptionExt, err := result.queryInscriptionExts()
	if err != nil {
		xylog.Logger.Fatalf("args %+v err %s", 1, err)
	}
	for _, ins := range inscriptionExt {
		result.tickExtMap.Store(strings.ToLower(ins.Tick), ins)
	}

	inscriptions, err := result.queryInscriptions()
	if err != nil {
		xylog.Logger.Fatalf("args %+v err %s", 1, err)
	}
	for _, inscription := range inscriptions {
		result.tickMap.Store(strings.ToLower(inscription.Tick), inscription)
	}
	if err != nil {
		xylog.Logger.Fatalf("args %+v err %s", 1, err)
	}
	for _, ins := range inscriptionExt {
		result.tickExtMap.Store(strings.ToLower(ins.Tick), ins)
	}
	addressMintTimes, err := result.queryAddressMintTimes()
	if err != nil {
		xylog.Logger.Fatalf("args %+v err %s", 1, err)
	}

	for _, v := range addressMintTimes {
		subMap, ok := result.allAddressMintTimesMap.Load(strings.ToLower(v.Tick))
		if !ok {
			subMap = &sync.Map{}
			result.allAddressMintTimesMap.Store(strings.ToLower(v.Tick), subMap)
		}
		t := subMap.(*sync.Map)
		t.Store(strings.ToLower(v.Address), v.MintTimes)
	}
	return result
}

func (p *Protocol) Parse(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	switch md.Operate {
	case OperateRegister:
		return p.ProcessRegister(block, tx, md)
	case devents.OperateDeploy:
		return p.ProcessDeploy(block, tx, md)
	case devents.OperateMint:
		return p.ProcessMint(block, tx, md)
	case devents.OperateTransfer:
		return p.ProcessTransfer(block, tx, md)
	case OperateList:
		return p.ProcessList(block, tx, md)
	case OperateBuy:
		return p.ProcessBuy(block, tx, md)
	case OperateProxyPay:
		return p.ProcessProxyPay(block, tx, md)
	}

	return nil, nil
}

func (p *Protocol) GetInscription(tickName string) *model.Inscriptions {
	insObj, ok := p.tickMap.Load(tickName)
	if !ok {
		inscription, err := p.queryInscription(tickName)
		if err != nil {
			xylog.Logger.Warnf(" settle %s mint not find tick", tickName)
			return nil
		}
		p.tickMap.Store(tickName, inscription)
		return inscription
	}
	return insObj.(*model.Inscriptions)
}

func (p *Protocol) GetInscriptionExt(tickName string) *model.OpbrcInscriptionExt {
	insExtObj, ok := p.tickExtMap.Load(tickName)
	if !ok {
		inscriptionExt := p.queryInscriptionExt(tickName)
		if inscriptionExt != nil {
			p.tickExtMap.Store(tickName, inscriptionExt)
			return inscriptionExt
		}
		xylog.Logger.Infof(" settle %s mint not find insExtObj", tickName)
		return nil
	}
	return insExtObj.(*model.OpbrcInscriptionExt)
}

func isValidEthAddress(address string) bool {
	ethAddressRegex := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return ethAddressRegex.MatchString(address)
}
