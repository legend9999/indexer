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

// Buy content eg {"p":"opbrc","op":"buy","tick":"obrc","list_tx":"0x536e887419e9a38c4756cebaff39fd65a0b3b8d71fd103a4833f486ab9762dcb","amt":"10000000","value":"750000000000000000","mp":"0x65efdd09dcbf0c6d769372dd07f8eb3f963f4a2d"}
type Buy struct {
	P      string          `json:"p"`
	Op     string          `json:"op"`
	Tick   string          `json:"tick"`
	ListTx string          `json:"list_tx"`
	Amt    decimal.Decimal `json:"amt"`
	Value  decimal.Decimal `json:"value"`
	Mp     string          `json:"mp"`
}

func (p *Protocol) ProcessBuy(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	_, err := p.checkBuy(block, tx, md)
	if err != nil {
		return nil, xyerrors.ErrDataVerifiedFailed.WrapCause(err)
	}
	return []*devents.TxResult{}, nil
}

func (p *Protocol) checkBuy(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) (*Buy, *xyerrors.InsError) {
	if strings.ToLower(tx.To) != strings.ToLower(p.cache.GlobalCfg.Chain.MarketPlaceAddress) {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("to address[%s] != mp address[%s]", tx.To, p.cache.GlobalCfg.Chain.MarketPlaceAddress))
	}
	// metadata protocol / tick checking
	if md.Protocol == "" || md.Tick == "" {
		return nil, xyerrors.NewInsError(-12, fmt.Sprintf("protocol[%s] / tick[%s] nil", md.Protocol, md.Tick))
	}

	// exists checking
	if ok, _ := p.cache.Inscription.Get(md.Protocol, md.Tick); !ok {
		return nil, xyerrors.NewInsError(-15, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s]", md.Protocol, md.Tick))
	}

	buy := &Buy{}
	err := json.Unmarshal([]byte(md.Data), buy)
	if err != nil {
		return nil, xyerrors.NewInsError(-13, fmt.Sprintf("json decode err:%v", err))
	}

	inscriptionExt := p.queryInscriptionExt(md.Tick)

	if inscriptionExt == nil {
		return nil, xyerrors.NewInsError(-11, fmt.Sprintf("tick[%s] not found", md.Tick))
	}
	if inscriptionExt.Progress != 1 {
		return nil, xyerrors.NewInsError(-10, fmt.Sprintf("tick[%s] progress[%d] != 1", md.Tick, inscriptionExt.Progress))
	}

	if err != nil {
		return nil, xyerrors.NewInsError(-13, fmt.Sprintf("json decode err:%v", err))
	}
	if buy.P != protocolName {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("protocol[%s] != protocol[%s]", buy.P, protocolName))
	}

	if strings.ToLower(buy.Mp) != strings.ToLower(p.cache.GlobalCfg.Chain.MarketPlaceAddress) {
		return nil, xyerrors.NewInsError(-16, fmt.Sprintf("op[%s] != op[%s]", buy.Op, p.cache.GlobalCfg.Chain.MarketPlaceAddress))
	}

	if buy.Amt.BigInt().Uint64() > math.MaxInt64 || buy.Amt.BigInt().Uint64() <= 0 {
		return nil, xyerrors.NewInsError(-17, fmt.Sprintf("amt[%v] > max int64", buy.Amt))
	}

	if buy.Value.BigInt().Uint64() > math.MaxInt64 || buy.Value.BigInt().Uint64() <= 0 {
		return nil, xyerrors.NewInsError(-18, fmt.Sprintf("value[%v] > max int64", buy.Value))
	}

	if tx.Value.Uint64() > math.MaxInt64 {
		return nil, xyerrors.NewInsError(-19, fmt.Sprintf("tx value[%v] > max int64", tx.Value))
	}

	if buy.Value.BigInt().Uint64() != tx.Value.Uint64() {
		return nil, xyerrors.NewInsError(-20, fmt.Sprintf("value[%v] != tx value[%v]", buy.Value, tx.Value))
	}

	dbClient := p.cache.GetDBClient().SqlDB

	mpTxEntity := &model.OpbrcMarketPlaceTx{
		Chain:           chainName,
		Protocol:        protocolName,
		Op:              md.Operate,
		Tick:            md.Tick,
		BlockNumber:     block.Number.Uint64(),
		TxHash:          strings.ToLower(tx.Hash),
		ListAddress:     "",
		BuyAddress:      strings.ToLower(tx.From),
		ProxyPayAddress: "",
		MpAddress:       strings.ToLower(buy.Mp),
		Amt:             buy.Amt,
		Value:           buy.Value,
		MDContent:       md.Data,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	result := dbClient.Create(mpTxEntity)
	if result.Error != nil {
		xylog.Logger.Warnf("update inscription ext err %s", result.Error)
		return nil, xyerrors.NewInsError(-23, fmt.Sprintf("update market place tx err %s", result.Error))
	}

	return buy, nil
}
