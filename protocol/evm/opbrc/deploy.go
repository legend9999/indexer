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

type Deploy struct {
	P      string          `json:"p"`
	Op     string          `json:"op"`
	Tick   string          `json:"tick"`
	Max    decimal.Decimal `json:"max"`
	Mspan  decimal.Decimal `json:"mspan"`
	Sm     decimal.Decimal `json:"sm"`
	Mcount decimal.Decimal `json:"mcount"`
	Cost   decimal.Decimal `json:"cost"`
	Dr     decimal.Decimal `json:"dr"`
}

func (p *Protocol) ProcessDeploy(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	d, err := p.checkDeploy(block, tx, md)
	if err != nil {
		return nil, xyerrors.ErrDataVerifiedFailed.WrapCause(err)
	}

	result := &devents.TxResult{
		MD:    md,
		Block: block,
		Tx:    tx,
		Deploy: &devents.Deploy{
			Name:      d.Tick,
			MaxSupply: d.Max,
			MintLimit: d.Mcount,
			Decimal:   int8(0),
		},
	}
	return []*devents.TxResult{result}, nil
}

func (p *Protocol) checkDeploy(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) (*Deploy, *xyerrors.InsError) {
	if strings.ToLower(tx.To) != strings.ToLower(p.cache.GlobalCfg.Chain.TreasuryAddress) {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("tx.to[%s] != treasury_address[%s]", tx.To, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}
	// metadata protocol / tick checking
	if md.Tick == "" || md.Protocol != protocolName {
		return nil, xyerrors.NewInsError(-210, fmt.Sprintf("protocol[%s] / tick[%s] nil", md.Protocol, md.Tick))
	}

	deploy := &Deploy{}
	err := json.Unmarshal([]byte(strings.ToLower(md.Data)), deploy)
	if err != nil {
		return nil, xyerrors.NewInsError(-211, fmt.Sprintf("json decode err:%v", err))
	}

	inscriptionExt := p.queryInscriptionExt(md.Tick)
	if inscriptionExt == nil {
		return nil, xyerrors.NewInsError(-212, fmt.Sprintf("tick %s not reg", deploy.Tick))
	}

	if strings.ToLower(inscriptionExt.RegistryAddress) != strings.ToLower(tx.From) {
		return nil, xyerrors.NewInsError(-213, fmt.Sprintf("tick %s reg addr %s not deply addr %s", deploy.Tick, inscriptionExt.RegistryAddress, tx.From))

	}

	// exists checking
	if ok, _ := p.cache.Inscription.Get(md.Protocol, md.Tick); ok {
		return nil, xyerrors.NewInsError(-15, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s]", md.Protocol, md.Tick))
	}

	// max > 0
	if deploy.Max.IntPart() > math.MaxInt64 || deploy.Max.IntPart() <= 0 {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s] max[%d]", md.Protocol, md.Tick, deploy.Max))
	}

	if deploy.Mspan.IntPart() > math.MaxInt32 || deploy.Mspan.IntPart() <= 0 {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s] mspan[%d]", md.Protocol, md.Tick, deploy.Mspan))
	}
	if deploy.Mcount.IntPart() > math.MaxInt32 || deploy.Mcount.IntPart() <= 0 {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s] mcount[%d]", md.Protocol, md.Tick, deploy.Mcount))
	}

	if deploy.Sm.IntPart() > math.MaxInt32 || deploy.Sm.IntPart() <= 0 {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s] sm[%d]", md.Protocol, md.Tick, deploy.Sm))
	}

	if deploy.Cost.IntPart() > math.MaxInt64 || deploy.Cost.IntPart() < 0 {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s] cost[%d]", md.Protocol, md.Tick, deploy.Cost))
	}

	//处理分成
	if deploy.Dr.GreaterThan(decimal.Zero) && deploy.Cost.GreaterThan(decimal.Zero) {
		//成功
	}
	if deploy.Dr.Equal(decimal.Zero) && deploy.Cost.Equal(decimal.Zero) {
		//成功
	}
	if deploy.Dr.GreaterThan(decimal.Zero) && deploy.Cost.LessThanOrEqual(decimal.Zero) || deploy.Dr.LessThan(decimal.Zero) && deploy.Cost.GreaterThanOrEqual(decimal.Zero) {
		//不成功
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("inscription deployed & abort, protocol[%s], tick[%s] dr and cost invalid [%+v]", md.Protocol, md.Tick, deploy))
	}

	if strings.ToLower(tx.To) != strings.ToLower(p.cache.GlobalCfg.Chain.TreasuryAddress) {
		return nil, xyerrors.NewInsError(-214, fmt.Sprintf("tick %s deply addr %s not treasury addr %s", deploy.Tick, tx.To, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}
	_, err = p.updateInscriptionExtOnDeploy(block, tx, *deploy)
	if err != nil {
		xylog.Logger.Warnf("update inscription ext on deploy err:%v", err)
		return nil, xyerrors.NewInsError(-211, fmt.Sprintf("update inscription ext on deploy err:%v", err))
	}
	return deploy, nil
}
