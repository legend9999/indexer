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

	"github.com/shopspring/decimal"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/devents"
	"github.com/uxuycom/indexer/xyerrors"
	"github.com/uxuycom/indexer/xylog"
)

type Transfer struct {
	P    string   `json:"p"`
	Op   string   `json:"op"`
	Tick string   `json:"tick"`
	To   []ToItem `json:"to"`
}

type ToItem struct {
	Recv string          `json:"recv"`
	Amt  decimal.Decimal `json:"amt"`
}

func (p *Protocol) ProcessTransfer(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	dTransfer, err := p.checkTransfer(tx, md)
	if err != nil {
		return nil, xyerrors.ErrDataVerifiedFailed.WrapCause(err)
	}

	result := &devents.TxResult{
		MD:    md,
		Block: block,
		Tx:    tx,
		Transfer: &devents.Transfer{
			Sender:   tx.From,
			Receives: make([]*devents.Receive, 0),
		},
	}
	for _, receive := range dTransfer.Receives {
		result.Transfer.Receives = append(result.Transfer.Receives, &devents.Receive{
			Address: receive.Address,
			Amount:  receive.Amount,
		})
	}
	return []*devents.TxResult{result}, nil
}

func (p *Protocol) checkTransfer(tx *xycommon.RpcTransaction, md *devents.MetaData) (*devents.Transfer, *xyerrors.InsError) {
	if strings.ToLower(tx.To) != strings.ToLower(p.cache.GlobalCfg.Chain.TreasuryAddress) {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("tx.to[%s] != treasury_address[%s]", tx.To, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}
	// metadata protocol / tick checking
	if md.Tick == "" || md.Protocol != protocolName {
		return nil, xyerrors.NewInsError(-210, fmt.Sprintf("protocol[%s] / tick[%s] nil", md.Protocol, md.Tick))
	}

	tf := &Transfer{}
	err := json.Unmarshal([]byte(md.Data), tf)
	if err != nil {
		return nil, xyerrors.NewInsError(-13, fmt.Sprintf("data json deocde err:%v, data[%s]", err, md.Data))
	}

	if len(tf.To) == 0 {
		return nil, xyerrors.NewInsError(-14, "to list is empty")
	}
	tempTos := make([]ToItem, 0)
	for _, to := range tf.To {
		if !isValidEthAddress(to.Recv) {
			xylog.Logger.Infof("to address is not valid eth address, to address[%s]", to.Recv)
			continue
		}
		if to.Amt.LessThanOrEqual(decimal.Zero) {
			xylog.Logger.Infof("to amount is zero, to amount[%v]", to.Amt)
			continue
		}
		if to.Amt.BigInt().Uint64() > math.MaxInt64 {
			xylog.Logger.Infof("to amount is too large, to amount[%v]", to.Amt)
			continue
		}
		tempTos = append(tempTos, to)
	}
	var (
		protocol = md.Protocol
		tick     = md.Tick
	)
	ok, inscription := p.cache.Inscription.Get(protocol, tick)
	if !ok || inscription == nil {
		return nil, xyerrors.NewInsError(-15, fmt.Sprintf("inscription not exist, protocol[%s]-tick[%s]", protocol, tick))
	}

	// sender balance checking
	ok, balance := p.cache.Balance.Get(protocol, tick, tx.From)
	if !ok {
		return nil, xyerrors.NewInsError(-16, fmt.Sprintf("sender balance record not exist, tick[%s-%s], address[%s]", protocol, tick, tx.From))
	}

	tempBalance := balance.Overall.Copy()

	result := &devents.Transfer{
		Sender:   tx.From,
		Receives: make([]*devents.Receive, 0),
	}
	for _, toItem := range tempTos {
		if tempBalance.LessThan(toItem.Amt) {
			continue
		}
		result.Receives = append(result.Receives, &devents.Receive{
			Address: toItem.Recv,
			Amount:  toItem.Amt,
		})
		tempBalance = tempBalance.Sub(toItem.Amt)
	}
	return result, nil
}
