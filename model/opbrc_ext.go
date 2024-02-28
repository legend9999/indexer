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

package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type OpbrcInscriptionExt struct {
	ID                  uint64          `gorm:"primaryKey" json:"id"`
	Chain               string          `json:"chain" gorm:"column:chain"`
	Protocol            string          `json:"protocol" gorm:"column:protocol"`
	Tick                string          `json:"tick" gorm:"column:tick"`
	RegistryAddress     string          `json:"registry_address" gorm:"column:registry_address"`
	RegistryBlockNumber uint64          `json:"registry_block_number" gorm:"column:registry_block_number"`
	Mspan               uint64          `json:"mspan" gorm:"column:mspan"`                               // mint总时间，小时，从deploy成功的块开始计算
	Cost                uint64          `json:"cost" gorm:"column:cost"`                                 // mint时最小的向国库付费金额
	OriginTick          string          `json:"origin_tick" gorm:"column:origin_tick"`                   // 原始tick，会区分大小写
	Mcount              uint64          `json:"mcount" gorm:"column:mcount"`                             // 单地址总的mint次数
	Sm                  uint64          `json:"sm" gorm:"column:sm"`                                     // settle minute 结算间隔，单位分钟
	Dr                  decimal.Decimal `json:"dr" gorm:"column:dr;type:decimal(38,18)"`                 // 分成比例
	SettledBlockNumber  uint64          `json:"settled_block_number" gorm:"column:settled_block_number"` // mint时已经结算的区块
	Progress            int             `json:"progress" gorm:"column:progress"`                         // 0=mint中，1=mint已结束
	EndBlockNumber      uint64          `json:"end_block_number" gorm:"column:end_block_number"`         // 结束区块
	StartBlockNumber    uint64          `json:"start_block_number" gorm:"column:start_block_number"`     // 开始区块，即部署区块
	AvgSettleQty        uint64          `json:"avg_settle_qty" gorm:"column:avg_settle_qty"`             // 每次结算的数量
	CreatedAt           time.Time       `json:"created_at" gorm:"column:created_at"`
	UpdatedAt           time.Time       `json:"updated_at" gorm:"column:updated_at"`
}

func (OpbrcInscriptionExt) TableName() string {
	return "opbrc_inscription_ext"
}

type OpbrcAddressMintTimes struct {
	ID                 uint64    `gorm:"primaryKey" json:"id"`
	Chain              string    `json:"chain" gorm:"column:chain"`
	Protocol           string    `json:"protocol" gorm:"column:protocol"`
	Address            string    `json:"address" gorm:"column:address;type:varchar(128)"`
	Tick               string    `json:"tick" gorm:"column:tick;type:varchar(512)"`
	MintTimes          uint64    `json:"mint_times" gorm:"column:mint_times"`
	CurrentSMMintTimes uint64    `json:"current_sm_mint_times" gorm:"column:current_sm_mint_times"`
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (OpbrcAddressMintTimes) TableName() string {
	return "opbrc_address_mint_times"
}

type OpbrcMarketPlaceTx struct {
	ID              uint64          `gorm:"primaryKey" json:"id"`
	Chain           string          `json:"chain" gorm:"column:chain"`
	Protocol        string          `json:"protocol" gorm:"column:protocol"`
	Op              string          `json:"op" gorm:"column:op"`
	Tick            string          `json:"tick" gorm:"column:tick"`
	BlockNumber     uint64          `json:"block_number" gorm:"column:block_number"`
	TxHash          string          `json:"tx_hash" gorm:"column:tx_hash"`
	ListAddress     string          `json:"list_address" gorm:"column:list_address"`
	BuyAddress      string          `json:"buy_address" gorm:"column:buy_address"`
	ProxyPayAddress string          `json:"proxy_pay_address" gorm:"column:proxy_pay_address"`
	MpAddress       string          `json:"mp_address" gorm:"column:mp_address"`
	Amt             decimal.Decimal `json:"amt" gorm:"column:amt;type:decimal(38,18)"`
	Value           decimal.Decimal `json:"value" gorm:"column:value;type:decimal(38,18)"`
	MDContent       string          `json:"md_content" gorm:"column:md_content"`
	ProcessStatus   int8            `json:"process_status" gorm:"column:process_status"`
	CreatedAt       time.Time       `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"column:updated_at"`
}

func (OpbrcMarketPlaceTx) TableName() string {
	return "opbrc_market_place_tx"
}

type OpbrcTempTxs struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Chain        string    `json:"chain" gorm:"column:chain"`                 // chain name
	Protocol     string    `json:"protocol" gorm:"column:protocol"`           // protocol name
	BlockHeight  uint64    `json:"block_height" gorm:"column:block_height"`   // block height
	FromAddress  string    `json:"from_address" gorm:"column:from_address"`   // from address
	TxHash       string    `json:"tx_hash" gorm:"column:tx_hash"`             // tx hash
	Op           string    `json:"op" gorm:"column:op"`                       // op code
	Tick         string    `json:"tick" gorm:"column:tick"`                   // inscription code
	BlockContent string    `json:"block_content" gorm:"column:block_content"` // inscription code
	TxContent    string    `json:"tx_content" gorm:"column:tx_content"`       // inscription code
	OpContent    string    `json:"op_content" gorm:"column:op_content"`       // inscription code
	MdContent    string    `json:"md_content" gorm:"column:md_content"`       // inscription code
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (OpbrcTempTxs) TableName() string {
	return "opbrc_temp_txs"
}
