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
	"time"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/devents"
	"github.com/uxuycom/indexer/model"
	"github.com/uxuycom/indexer/xyerrors"
)

type Register struct {
	P    string `json:"p"`
	Op   string `json:"op"`
	Tick string `json:"tick"`
}

func (p *Protocol) ProcessRegister(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) ([]*devents.TxResult, *xyerrors.InsError) {
	_, err := p.checkRegister(block, tx, md)
	if err != nil {
		return nil, xyerrors.ErrDataVerifiedFailed.WrapCause(err)
	}
	return nil, nil
}

func (p *Protocol) checkRegister(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, md *devents.MetaData) (*Register, *xyerrors.InsError) {
	if tx.To != p.cache.GlobalCfg.Chain.TreasuryAddress {
		return nil, xyerrors.NewInsError(-14, fmt.Sprintf("tx.to[%s] != treasury_address[%s]", tx.To, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}
	// metadata protocol / tick checking
	if md.Protocol == "" || md.Tick == "" {
		return nil, xyerrors.NewInsError(-200, fmt.Sprintf("protocol[%s] / tick[%s] nil", md.Protocol, md.Tick))
	}
	var register Register
	if err := json.Unmarshal([]byte(md.Data), &register); err != nil {
		return nil, xyerrors.NewInsError(-201, fmt.Sprintf("protocol[%s] / data [%s] unmarshal err %+v", md.Protocol, md.Data, err))
	}

	register.Tick = strings.TrimSpace(register.Tick)
	if len(register.Tick) == 0 {
		return nil, xyerrors.NewInsError(-202, fmt.Sprintf("protocol[%s] / register tick[%s] nil", md.Protocol, md.Tick))
	}
	if len(register.Tick) < 2 {
		return nil, xyerrors.NewInsError(-202, fmt.Sprintf("protocol[%s] / register tick[%s] length less than 2", md.Protocol, md.Tick))
	}

	registerFee := int64(0)
	if len(register.Tick) >= 5 {
		registerFee = p.cache.GlobalCfg.Chain.RegisterFee[5].Shift(18).BigInt().Int64()
	} else {
		registerFee = p.cache.GlobalCfg.Chain.RegisterFee[len(register.Tick)].Shift(18).BigInt().Int64()
	}

	if tx.Value.Int64() < registerFee {
		return nil, xyerrors.NewInsError(-203, fmt.Sprintf("protocol[%s] / register tick[%s] register fee[%d] less than tx value[%d]", md.Protocol, md.Tick, registerFee, tx.Value.Int64()))
	}
	if strings.ToLower(tx.To) != strings.ToLower(p.cache.GlobalCfg.Chain.TreasuryAddress) {
		return nil, xyerrors.NewInsError(-204, fmt.Sprintf("protocol[%s] / register tick[%s] register address[%s] not equal to treasury address[%s]", md.Protocol, md.Tick, tx.From, p.cache.GlobalCfg.Chain.TreasuryAddress))
	}

	dbClient := p.cache.GetDBClient().SqlDB
	inscriptionExt := p.queryInscriptionExt(register.Tick)
	if inscriptionExt != nil {
		return nil, xyerrors.NewInsError(-202, fmt.Sprintf("protocol[%s] / register tick[%s] exsist", md.Protocol, md.Tick))
	}
	insExt := &model.OpbrcInscriptionExt{
		Chain:               chainName,
		Protocol:            protocolName,
		Tick:                strings.ToLower(register.Tick),
		RegistryAddress:     strings.ToLower(tx.From),
		RegistryBlockNumber: tx.BlockNumber.Uint64(),
		OriginTick:          register.Tick,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	if err := dbClient.Create(insExt).Error; err != nil {
		return nil, xyerrors.NewInsError(-203, fmt.Sprintf("protocol[%s] / tick[%s] save err %+v", md.Protocol, md.Tick, err))
	}

	return &register, nil
}
