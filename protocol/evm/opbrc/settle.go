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
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/shopspring/decimal"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/devents"
	"github.com/uxuycom/indexer/model"
	"github.com/uxuycom/indexer/xyerrors"
	"github.com/uxuycom/indexer/xylog"
)

type tempSettleMint struct {
	Block *xycommon.RpcBlock
	Tx    *xycommon.RpcTransaction
	Mint  *Mint
	Md    *devents.MetaData
}

func (p *Protocol) Settle(block *xycommon.RpcBlock) ([]*devents.TxResult, *xyerrors.InsError) {
	inscriptionExts := p.notSettledTick()
	result := make([]*devents.TxResult, 0)
	for _, inscriptionExt := range inscriptionExts {
		txResults, insError := p.settle(block, inscriptionExt)
		if insError != nil {
			xylog.Logger.Warnf("tick %s settle error: %s", inscriptionExt.Tick, insError.Error())
			continue
		}
		result = append(result, txResults...)
	}
	return result, nil
}

// During the settlement cycle, update the settlement data of opbrc
func (p *Protocol) settle(block *xycommon.RpcBlock, insExt *model.OpbrcInscriptionExt) ([]*devents.TxResult, *xyerrors.InsError) {
	tickName := strings.ToLower(insExt.Tick)
	ins := p.GetInscription(tickName)
	if ins == nil {
		return nil, xyerrors.NewInsError(-101, fmt.Sprintf("tick %s not found", tickName))
	}

	totalSupply := ins.TotalSupply                  // 总发行量
	mspan := insExt.Mspan                           // Mint 总时间
	cost := insExt.Cost                             // 付费mint的bnb量，wei
	mcount := insExt.Mcount                         // 单地址mint总次数
	endBlockNumber := insExt.EndBlockNumber         // 结束区块
	settledBlockNumber := insExt.SettledBlockNumber // 已结算的区块
	avgSettleQty := insExt.AvgSettleQty             // 平均结算量
	progress := insExt.Progress                     // 进度
	sm := insExt.Sm                                 // 结算周期（分钟）

	// 判断是否到结算的时间点
	if settledBlockNumber >= endBlockNumber {
		xylog.Logger.Warnf("settle tick %s endBlockNumber = %d parsedMaxBlockNumber = %d", tickName, endBlockNumber, block.Number.Uint64())
		return nil, xyerrors.NewInsError(-102, fmt.Sprintf("tick %s endBlockNumber = %d parsedMaxBlockNumber = %d", tickName, endBlockNumber, block.Number.Uint64()))
	}
	toBlockNumber := settledBlockNumber + sm*60
	if toBlockNumber > block.Number.Uint64() {
		xylog.Logger.Debugf("settle tick %s toBlockNumber = %d parsedMaxBlockNumber = %d", tickName, toBlockNumber, block.Number.Uint64())
		return nil, xyerrors.NewInsError(-103, fmt.Sprintf("tick %s toBlockNumber = %d parsedMaxBlockNumber = %d", tickName, toBlockNumber, block.Number.Uint64()))
	}
	xylog.Logger.Infof(" settle tick %s totalSupply = %s mspan = %d cost = %d mcount = %d endBlockNumber = %d settledBlockNumber = %d avgSettleQty = %d progress = %d sm = %d",
		tickName, totalSupply, mspan, cost, mcount, endBlockNumber, settledBlockNumber, avgSettleQty, progress, sm)

	tickMintTxsObj, ok := p.allAddressCurrentSmMintTxMap.Load(tickName)
	if !ok {
		xylog.Logger.Infof(" settle %s mint addr is empty auto settle to depolyer", tickName)
		tickMintTxsObj = make([]*tempSettleMint, 0)
	}
	tickMintTxs := tickMintTxsObj.([]*tempSettleMint)

	xylog.Logger.Infof(" settle %s size %d", tickName, len(tickMintTxs))

	//第一次mint，需要插入
	newAddrMint := make(map[string]struct{})

	validTickMintTx := make([]*tempSettleMint, 0)
	//本次结算有效果mint总次数
	validTimes := 0
	for _, tickMintTx := range tickMintTxs {
		minter := strings.ToLower(tickMintTx.Tx.From)
		mintedTimes := p.getMintTimes(tickName, minter)
		if mintedTimes == 0 {
			newAddrMint[minter] = struct{}{}
		}
		if mintedTimes >= mcount {
			p.incrMintTimes(tickName, minter, 1)
			continue
		}
		p.incrMintTimes(tickName, minter, 1)
		validTimes += 1
		validTickMintTx = append(validTickMintTx, tickMintTx)
	}
	mintAmtResult := make([]*devents.TxResult, 0)
	// 没有minter，全给deploy
	if validTimes == 0 {
		mintAmtResult = append(mintAmtResult, &devents.TxResult{
			MD: &devents.MetaData{
				Chain:    chainName,
				Protocol: protocolName,
				Operate:  devents.OperateMint,
				Tick:     tickName,
				Data:     "",
			},
			Block: block,
			Tx: &xycommon.RpcTransaction{
				From:        ins.DeployBy,
				BlockNumber: block.Number,
				TxIndex:     big.NewInt(0),
				Hash:        fmt.Sprintf("auto_settle_%d", block.Number.Uint64()),
				To:          ins.DeployBy,
				Gas:         big.NewInt(0),
				GasPrice:    big.NewInt(0),
			},
			Mint: &devents.Mint{
				Minter: ins.DeployBy,
				Amount: decimal.RequireFromString(fmt.Sprintf("%d", avgSettleQty)),
			},
			Deploy:   nil,
			Transfer: nil,
		})
	} else {
		//单次mint分配的量
		singleMintQty := avgSettleQty / uint64(validTimes)
		xylog.Logger.Debugf("singleMintQty %d", singleMintQty)

		for _, settleMint := range validTickMintTx {
			mintAmtResult = append(mintAmtResult, &devents.TxResult{
				MD:    settleMint.Md,
				Block: settleMint.Block,
				Tx:    settleMint.Tx,
				Mint: &devents.Mint{
					Minter: settleMint.Tx.From,
					Amount: decimal.NewFromInt(int64(singleMintQty)),
				},
				Deploy:   nil,
				Transfer: nil,
			})
		}
	}

	//mint结束
	if block.Number.Uint64() >= endBlockNumber {
		//insExt.Progress = 1
		err := p.updateProgressMintFinish(p.cache.GetDBClient().SqlDB, tickName)
		if err != nil {
			xylog.Logger.Warnf("tick %s updateProgressMintFinish err %s", tickName, err)
		}
		//mintedQty, err := p.mintedQty(p.cache.GetDBClient().SqlDB, tickName)
		ok, stats := p.cache.InscriptionStats.Get(protocolName, tickName)
		if !ok {
			xylog.Logger.Warnf("tick %s mintedQty err %s", tickName, err)
		} else {
			//剩下的全给deploy
			leftQty := totalSupply.Sub(stats.Minted)
			//减去本次结算的量
			for _, mar := range mintAmtResult {
				leftQty = leftQty.Sub(mar.Mint.Amount)
			}
			mintAmtResult = append(mintAmtResult, &devents.TxResult{
				MD: &devents.MetaData{
					Chain:    chainName,
					Protocol: protocolName,
					Operate:  devents.OperateMint,
					Tick:     tickName,
					Data:     "",
				},
				Block: block,
				Tx: &xycommon.RpcTransaction{
					From:        ins.DeployBy,
					BlockNumber: block.Number,
					TxIndex:     big.NewInt(0),
					Hash:        fmt.Sprintf("mint_finish_auto_settle_%d", block.Number.Uint64()),
					To:          ins.DeployBy,
					Gas:         big.NewInt(0),
					GasPrice:    big.NewInt(0),
				},
				Mint: &devents.Mint{
					Minter: ins.DeployBy,
					Amount: leftQty,
				},
				Deploy:   nil,
				Transfer: nil,
			})
		}
	}

	//更新mint次数
	{
		needInsertAddrMintTimes := make(map[string]uint64)
		needUpdateAddrMintTimes := make(map[string]uint64)
		p.getTickAllAddrMintTimes(tickName).Range(func(key, value any) bool {
			_, ok := newAddrMint[key.(string)]
			if ok {
				needInsertAddrMintTimes[key.(string)] = value.(uint64)
			} else {
				needUpdateAddrMintTimes[key.(string)] = value.(uint64)
			}
			return true
		})
		_, err := p.insertMintTimes(tickName, needInsertAddrMintTimes)
		if err != nil {
			xylog.Logger.Warnf("insertMintTimes err %s", err)
		}
		_, err = p.updateMintTimes(tickName, needUpdateAddrMintTimes)
		if err != nil {
			xylog.Logger.Warnf("updateMintTimes err %s", err)
		}

	}

	// 更新SettledBlockNumber
	err := p.updateSettledBlockNumber(p.cache.GetDBClient().SqlDB, tickName, block.Number.Uint64())
	if err != nil {
		xylog.Logger.Warnf("updateSettledBlockNumber err %s", err)
		return nil, xyerrors.NewInsError(-100, "updateSettledBlockNumber err")
	}

	//删除本次结算完的数据
	p.allAddressCurrentSmMintTxMap.Delete(tickName)
	_, err = p.deleteTempTx(tickName, settledBlockNumber+1, toBlockNumber)
	if err != nil {
		xylog.Logger.Warnf("deleteTempTx tick [%s] err %s", tickName, err)
	}

	return mintAmtResult, nil

}

func (p *Protocol) getTickAllAddrMintTimes(tick string) *sync.Map {
	tick = strings.ToLower(tick)
	tickAddrMintTimesMapObj, _ := p.allAddressMintTimesMap.Load(tick)
	if tickAddrMintTimesMapObj == nil {
		tickAddrMintTimesMapObj = &sync.Map{}
		p.allAddressMintTimesMap.Store(tick, tickAddrMintTimesMapObj)
	}
	tickAddrMintTimesMap := tickAddrMintTimesMapObj.(*sync.Map)
	return tickAddrMintTimesMap
}
func (p *Protocol) getMintTimes(tick, minter string) uint64 {
	tick = strings.ToLower(tick)
	tickAddrMintTimesMapObj, _ := p.allAddressMintTimesMap.Load(tick)
	if tickAddrMintTimesMapObj == nil {
		tickAddrMintTimesMapObj = &sync.Map{}
		p.allAddressMintTimesMap.Store(tick, tickAddrMintTimesMapObj)
	}
	tickAddrMintTimesMap := tickAddrMintTimesMapObj.(*sync.Map)
	value, ok := tickAddrMintTimesMap.Load(strings.ToLower(minter))
	if !ok {
		return 0
	}
	return value.(uint64)
}

func (p *Protocol) incrMintTimes(tick, minter string, times uint64) {
	tick = strings.ToLower(tick)
	tickAddrMintTimesMapObj, _ := p.allAddressMintTimesMap.Load(tick)
	if tickAddrMintTimesMapObj == nil {
		tickAddrMintTimesMapObj = &sync.Map{}
	}
	tickAddrMintTimesMap := tickAddrMintTimesMapObj.(*sync.Map)
	value, ok := tickAddrMintTimesMap.Load(strings.ToLower(minter))
	if !ok {
		tickAddrMintTimesMap.Store(strings.ToLower(minter), times)
		return
	}
	tickAddrMintTimesMap.Store(strings.ToLower(minter), value.(uint64)+times)
}
