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
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/devents"
	"github.com/uxuycom/indexer/model"
	"github.com/uxuycom/indexer/xyerrors"
	"github.com/uxuycom/indexer/xylog"
)

// List content eg {"p":"opbrc","op":"list","tick":"obrc","amt":"100000000","value":"3150000000000000000","mp":"0x65efdd09dcbf0c6d769372dd07f8eb3f963f4a2d"}
type List struct {
	P     string          `json:"p"`
	Op    string          `json:"op"`
	Tick  string          `json:"tick"`
	Amt   decimal.Decimal `json:"amt"`
	Value decimal.Decimal `json:"value"`
	Mp    string          `json:"mp"`
}

func (p *Protocol) ProcessList(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	_, err := p.checkList(block, tx, md)
	if err != nil {
		return nil, xyerrors.ErrDataVerifiedFailed.WrapCause(err)
	}
	return []*devents.TxResult{}, nil
}

func (p *Protocol) checkList(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) (*List, *xyerrors.InsError) {
	if strings.ToLower(tx.To) != p.cache.GlobalCfg.Chain.TreasuryAddress {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("tx.to[%s] != treasury_address[%s]", tx.To, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}
	// metadata protocol / tick checking
	if md.Tick == "" || md.Protocol != protocolName {
		return nil, xyerrors.NewInsError(-210, fmt.Sprintf("protocol[%s] / tick[%s] nil", md.Protocol, md.Tick))
	}

	// exists checking
	if ok, _ := p.cache.Inscription.Get(md.Protocol, md.Tick); !ok {
		return nil, xyerrors.NewInsError(-15, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s]", md.Protocol, md.Tick))
	}

	inscriptionExt := p.queryInscriptionExt(md.Tick)

	if inscriptionExt == nil {
		return nil, xyerrors.NewInsError(-11, fmt.Sprintf("tick[%s] not found", md.Tick))
	}
	if inscriptionExt.Progress != 1 {
		return nil, xyerrors.NewInsError(-10, fmt.Sprintf("tick[%s] progress[%d] != 1", md.Tick, inscriptionExt.Progress))
	}

	list := &List{}
	err := json.Unmarshal([]byte(strings.ToLower(md.Data)), list)
	if err != nil {
		return nil, xyerrors.NewInsError(-13, fmt.Sprintf("json decode err:%v", err))
	}
	if list.P != protocolName {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("protocol[%s] != protocol[%s]", list.P, protocolName))
	}

	if strings.ToLower(list.Mp) != strings.ToLower(p.cache.GlobalCfg.Chain.MarketPlaceAddress) {
		return nil, xyerrors.NewInsError(-16, fmt.Sprintf("op[%s] != op[%s]", list.Op, p.cache.GlobalCfg.Chain.MarketPlaceAddress))
	}

	if list.Amt.BigInt().Uint64() > math.MaxInt64 || list.Amt.BigInt().Uint64() <= 0 {
		return nil, xyerrors.NewInsError(-17, fmt.Sprintf("amt[%v] > max int64", list.Amt))
	}

	if list.Value.BigInt().Uint64() > math.MaxInt64 || list.Value.BigInt().Uint64() <= 0 {
		return nil, xyerrors.NewInsError(-18, fmt.Sprintf("value[%v] > max int64", list.Value))
	}

	if tx.Value.Uint64() > math.MaxInt64 {
		return nil, xyerrors.NewInsError(-19, fmt.Sprintf("tx value[%v] > max int64", tx.Value))
	}

	ok, balanceItem := p.cache.Balance.Get(md.Protocol, md.Tick, tx.From)
	if !ok {
		return nil, xyerrors.NewInsError(-21, fmt.Sprintf("balance not found, protocol[%s], tick[%s], address[%s]", md.Protocol, md.Tick, tx.From))
	}

	if balanceItem.Overall.LessThan(list.Amt) {
		return nil, xyerrors.NewInsError(-22, fmt.Sprintf("balance not enough, protocol[%s], tick[%s], address[%s], available[%v], required[%v]", md.Protocol, md.Tick, tx.From, balanceItem.Available, list.Amt))
	}

	dbClient := p.cache.GetDBClient().SqlDB

	mpTxEntity := &model.OpbrcMarketPlaceTx{
		Chain:           chainName,
		Protocol:        protocolName,
		Op:              md.Operate,
		Tick:            md.Tick,
		BlockNumber:     block.Number.Uint64(),
		TxHash:          strings.ToLower(tx.Hash),
		ListAddress:     strings.ToLower(tx.From),
		BuyAddress:      "",
		ProxyPayAddress: "",
		MpAddress:       strings.ToLower(list.Mp),
		Amt:             list.Amt,
		Value:           list.Value,
		MDContent:       md.Data,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	result := dbClient.Create(mpTxEntity)
	if result.Error != nil {
		xylog.Logger.Warnf("update inscription ext err %s", result.Error)
		return nil, xyerrors.NewInsError(-23, fmt.Sprintf("update market place tx err %s", result.Error))
	}

	return list, nil
}
