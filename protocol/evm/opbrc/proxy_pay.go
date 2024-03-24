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

// ProxyPay content eg {"p":"opbrc","op":"proxy_pay","mp":"0x65efdd09dcbf0c6d769372dd07f8eb3f963f4a2d","tick":"obrc","amt":"20000000","value":"1500000000000000000","fee":"30000000000000000.00","list_tx":"0xab745ca50a2c8799b9e769a51a5f5e8335cd33633361ffaa46a4a6961b9d06cc","buy_tx":"0x95710541a80befcff4606fdf75dd69a50789424e84fa2c1a0c384d5957ac4d2a","pt_tx":"0x05f81e31c1c865035667b7057aef7da337271ff324e93c54543b85c5d9ac03be"}
type ProxyPay struct {
	P      string          `json:"p"`
	Op     string          `json:"op"`
	Mp     string          `json:"mp"`
	Tick   string          `json:"tick"`
	Amt    decimal.Decimal `json:"amt"`
	Value  decimal.Decimal `json:"value"`
	Fee    decimal.Decimal `json:"fee"`
	ListTx string          `json:"list_tx"`
	BuyTx  string          `json:"buy_tx"`
	PtTx   string          `json:"pt_tx"`
}

func (p *Protocol) ProcessProxyPay(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	transfer, err := p.checkProxyPay(block, tx, md)
	if err != nil {
		return nil, xyerrors.ErrDataVerifiedFailed.WrapCause(err)
	}

	result := &devents.TxResult{
		MD:       md,
		Block:    block,
		Tx:       tx,
		Transfer: transfer,
	}
	return []*devents.TxResult{result}, nil
}

func (p *Protocol) checkProxyPay(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) (*devents.Transfer, *xyerrors.InsError) {
	if strings.ToLower(tx.From) != strings.ToLower(p.cache.GlobalCfg.Chain.MarketPlaceAddress) {
		return nil, xyerrors.NewInsError(-11, fmt.Sprintf("tx.From[%s] != mpAddress[%s]", tx.From, p.cache.GlobalCfg.Chain.MarketPlaceAddress))
	}
	// metadata protocol / tick checking
	if md.Tick == "" || md.Protocol != protocolName {
		return nil, xyerrors.NewInsError(-210, fmt.Sprintf("protocol[%s] / tick[%s] nil", md.Protocol, md.Tick))
	}

	// exists checking
	if ok, _ := p.cache.Inscription.Get(md.Protocol, md.Tick); !ok {
		return nil, xyerrors.NewInsError(-15, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s]", md.Protocol, md.Tick))
	}

	proxyPay := &ProxyPay{}
	err := json.Unmarshal([]byte(strings.ToLower(md.Data)), proxyPay)
	if err != nil {
		return nil, xyerrors.NewInsError(-13, fmt.Sprintf("json decode err:%v", err))
	}

	if proxyPay.P != protocolName {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("protocol[%s] != protocol[%s]", proxyPay.P, protocolName))
	}

	if strings.ToLower(proxyPay.Mp) != strings.ToLower(p.cache.GlobalCfg.Chain.MarketPlaceAddress) {
		return nil, xyerrors.NewInsError(-16, fmt.Sprintf("op[%s] != op[%s]", proxyPay.Op, p.cache.GlobalCfg.Chain.MarketPlaceAddress))
	}

	if proxyPay.Amt.BigInt().Uint64() > math.MaxInt64 || proxyPay.Amt.BigInt().Uint64() <= 0 {
		return nil, xyerrors.NewInsError(-17, fmt.Sprintf("amt[%v] > max int64", proxyPay.Amt))
	}

	if proxyPay.Value.BigInt().Uint64() > math.MaxInt64 || proxyPay.Value.BigInt().Uint64() <= 0 {
		return nil, xyerrors.NewInsError(-18, fmt.Sprintf("value[%v] > max int64", proxyPay.Value))
	}

	if tx.Value.Uint64() > math.MaxInt64 {
		return nil, xyerrors.NewInsError(-19, fmt.Sprintf("tx value[%v] > max int64", tx.Value))
	}

	dbClient := p.cache.GetDBClient().SqlDB

	processStatus := 1

	listTx := model.OpbrcMarketPlaceTx{}
	err = dbClient.Table(model.OpbrcMarketPlaceTx{}.TableName()).Where("tx_hash = ?", strings.ToLower(proxyPay.ListTx)).Find(&listTx).Error
	if err != nil {
		xylog.Logger.Warnf("list tx [%s] not found, err %s", proxyPay.ListTx, err)
		processStatus = 0
	}

	buyTx := model.OpbrcMarketPlaceTx{}
	err = dbClient.Table(model.OpbrcMarketPlaceTx{}.TableName()).Where("tx_hash = ?", strings.ToLower(proxyPay.BuyTx)).Find(&buyTx).Error
	if err != nil {
		xylog.Logger.Warnf("buyTx [%s] not found, err %s", proxyPay.BuyTx, err)
		processStatus = 0
	}

	mpTxEntity := &model.OpbrcMarketPlaceTx{
		Chain:           chainName,
		Protocol:        protocolName,
		Op:              md.Operate,
		Tick:            md.Tick,
		BlockNumber:     block.Number.Uint64(),
		TxHash:          strings.ToLower(tx.Hash),
		ListAddress:     "",
		BuyAddress:      "",
		ProxyPayAddress: "",
		MpAddress:       strings.ToLower(proxyPay.Mp),
		Amt:             proxyPay.Amt,
		Value:           proxyPay.Value,
		MDContent:       md.Data,
		ProcessStatus:   int8(processStatus),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	result := dbClient.Create(mpTxEntity)
	if result.Error != nil {
		xylog.Logger.Warnf("update inscription ext err %s", result.Error)
		return nil, xyerrors.NewInsError(-23, fmt.Sprintf("update market place tx err %s", result.Error))
	}

	if processStatus == 0 {
		return nil, xyerrors.NewInsError(-22, fmt.Sprintf("listTx [%s] / buyTx [%s] not found", proxyPay.ListTx, proxyPay.BuyTx))
	}
	// sender balance checking
	ok, balance := p.cache.Balance.Get(md.Protocol, md.Tick, listTx.ListAddress)
	if !ok {
		return nil, xyerrors.NewInsError(-16, fmt.Sprintf("sender balance record not exist, tick[%s-%s], address[%s]", md.Protocol, md.Tick, tx.From))
	}

	if balance.Overall.LessThan(proxyPay.Amt) {
		return nil, xyerrors.NewInsError(-17, fmt.Sprintf("sender balance not enough, tick[%s-%s], address[%s], balance[%v], amt[%v]", md.Protocol, md.Tick, tx.From, balance.Overall, proxyPay.Amt))
	}
	return &devents.Transfer{
		Sender: listTx.ListAddress,
		Receives: []*devents.Receive{{
			Address: buyTx.BuyAddress,
			Amount:  proxyPay.Amt,
		},
		},
	}, nil
}
