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
	"strings"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/devents"
	"github.com/uxuycom/indexer/xyerrors"
	"github.com/uxuycom/indexer/xylog"
)

type Mint struct {
	P    string `json:"p"`
	Op   string `json:"op"`
	Tick string `json:"tick"`
}

func (p *Protocol) ProcessMint(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	_, err := p.checkMint(block, tx, md)
	if err != nil {
		xylog.Logger.Warnf("mint check err:%v, data[%s]", err, md.Data)
		//这里不能返回错误，否则同一区块的tx，如果一个错误，那么会导致整个区块都失败
		return nil, nil
	}
	return nil, nil
}

func (p *Protocol) checkMint(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) (*Mint, *xyerrors.InsError) {
	if strings.ToLower(tx.To) != strings.ToLower(p.cache.GlobalCfg.Chain.TreasuryAddress) {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("tx.to[%s] != treasury_address[%s]", tx.To, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}
	//blockNumber := tx.BlockNumber.Int64()

	mint := &Mint{}
	err := json.Unmarshal([]byte(md.Data), mint)
	if err != nil {
		return nil, xyerrors.NewInsError(-13, fmt.Sprintf("data json deocde err:%v, data[%s]", err, md.Data))
	}

	inscription := p.GetInscription(mint.Tick)
	if inscription == nil {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("tick[%s] not found", mint.Tick))
	}
	inscriptionExt := p.GetInscriptionExt(mint.Tick)
	if inscriptionExt == nil {
		return nil, xyerrors.NewInsError(-15, fmt.Sprintf("tickExt[%s] not found", mint.Tick))
	}
	if block.Number.Int64() > int64(inscriptionExt.EndBlockNumber) {
		return nil, xyerrors.NewInsError(-16, fmt.Sprintf("tick[%s] end block number[%d] > current block number[%d]", mint.Tick, inscriptionExt.EndBlockNumber, block.Number.Int64()))
	}

	if tx.Value.Int64() < int64(inscriptionExt.Cost) {
		return nil, xyerrors.NewInsError(-17, fmt.Sprintf("mint cost[%d] > tx value[%d]", inscriptionExt.Cost, tx.Value.Int64()))
	}

	if strings.ToLower(tx.To) != strings.ToLower(p.cache.GlobalCfg.Chain.TreasuryAddress) {
		return nil, xyerrors.NewInsError(-18, fmt.Sprintf("tx.to[%s] != inscription.Ext.RegistryAddress[%s]", tx.To, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}

	//mintTimes := p.getMintTimes(md.Tick, tx.From)
	//if mintTimes >= inscriptionExt.Mcount {
	//	xylog.Logger.Infof("addr [%s] mint times[%d] >= inscriptionExt.Mcount[%d]", tx.From, mintTimes, inscriptionExt.Mcount)
	//	return nil, xyerrors.NewInsError(-19, fmt.Sprintf("mint times[%d] >= inscriptionExt.Mcount[%d]", mintTimes, inscriptionExt.Mcount))
	//}

	//把mint事件暂存
	tickMintTxsObj, ok := p.allAddressCurrentSmMintTxMap.Load(mint.Tick)
	if !ok {
		tickMintTxsObj = make([]*tempSettleMint, 0)
		p.allAddressCurrentSmMintTxMap.Store(mint.Tick, tickMintTxsObj)
	}
	tickMintTxs := tickMintTxsObj.([]*tempSettleMint)
	temp := &tempSettleMint{
		Block: block,
		Tx:    tx,
		Mint:  mint,
		Md:    md,
	}
	tickMintTxs = append(tickMintTxs, temp)

	p.allAddressCurrentSmMintTxMap.Store(mint.Tick, tickMintTxs)

	//_, err = p.insertTempTx(temp)
	//if err != nil {
	//	xylog.Logger.Warnf("mint insert err:%v, data[%s]", err, md.Data)
	//}
	return mint, nil
}
