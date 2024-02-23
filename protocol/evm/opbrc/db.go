package opbrc

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/uxuycom/indexer/client/xycommon"
	"github.com/uxuycom/indexer/model"
	"github.com/uxuycom/indexer/xylog"
)

// 更新tick 为 mint 完成
func (p *Protocol) updateProgressMintFinish(tx *gorm.DB, tick string) error {
	err := tx.Table(model.OpbrcInscriptionExt{}.TableName()).Where(" tick = ?", tick).Update("progress", 1).Error
	if err != nil {
		xylog.Logger.Warnf("args %+v err %s", tick, err)
	}
	return err
}

// 结算后更新结算块
func (p *Protocol) updateSettledBlockNumber(tx *gorm.DB, tick string, settledBlockNumber uint64) error {
	err := tx.Table(model.OpbrcInscriptionExt{}.TableName()).Where(" tick = ?", tick).Update("settled_block_number", settledBlockNumber).Error
	if err != nil {
		xylog.Logger.Warnf("args %+v err %s", tick, err)
	}
	return err
}

// 已mint总量
func (p *Protocol) mintedQty(tx *gorm.DB, tick string) (decimal.Decimal, error) {
	var minted decimal.Decimal
	result := tx.Table(model.Balances{}.TableName()).Where("tick = ?", tick).Where("protocol = ?", protocolName).Where("chain = ?", chainName).
		Select("sum(available) + sum(balance)").Scan(&minted)
	if result.Error != nil {
		xylog.Logger.Warnf("select mint qty err %s", result.Error)
		return decimal.Zero, result.Error
	}
	return minted, nil
}

func (p *Protocol) queryInscriptionExts() ([]*model.OpbrcInscriptionExt, error) {
	dbClient := p.cache.GetDBClient().SqlDB
	inscriptionExt := make([]*model.OpbrcInscriptionExt, 0)
	if err := dbClient.Table(model.OpbrcInscriptionExt{}.TableName()).Find(&inscriptionExt).Error; err != nil {
		return nil, err
	}
	return inscriptionExt, nil
}

func (p *Protocol) queryAddressMintTimes() ([]*model.OpbrcAddressMintTimes, error) {
	dbClient := p.cache.GetDBClient().SqlDB
	inscriptionExt := make([]*model.OpbrcAddressMintTimes, 0)
	if err := dbClient.Table(model.OpbrcAddressMintTimes{}.TableName()).Find(&inscriptionExt).Error; err != nil {
		return nil, err
	}
	return inscriptionExt, nil
}

func (p *Protocol) queryInscriptionExt(tick string) *model.OpbrcInscriptionExt {
	dbClient := p.cache.GetDBClient().SqlDB
	var insExt *model.OpbrcInscriptionExt
	err := dbClient.Where("tick = ?", strings.ToLower(tick)).First(&insExt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || err != nil {
		return nil
	}
	return insExt
}

func (p *Protocol) queryInscriptions() ([]*model.Inscriptions, error) {
	dbClient := p.cache.GetDBClient().SqlDB
	inscriptions := make([]*model.Inscriptions, 0)
	if err := dbClient.Table(model.Inscriptions{}.TableName()).Find(&inscriptions).Error; err != nil {
		return nil, err
	}
	return inscriptions, nil
}
func (p *Protocol) queryInscription(tick string) (*model.Inscriptions, error) {
	dbClient := p.cache.GetDBClient().SqlDB
	var inscriptions *model.Inscriptions
	if err := dbClient.Table(model.Inscriptions{}.TableName()).Where("tick = ?", tick).First(&inscriptions).Error; err != nil {
		return nil, err
	}
	return inscriptions, nil
}

func (p *Protocol) updateMintTimes(tick string, mintTimes map[string]uint64) (int64, error) {
	if len(mintTimes) == 0 {
		return 0, nil
	}
	dbClient := p.cache.GetDBClient().SqlDB
	sqlPrefix := `UPDATE opbrc_address_mint_times
					SET mint_times =
							CASE`
	sqlVar := `WHEN address = '%s' THEN %d`
	sqlSuffix := `
			ELSE mint_times
			END
			WHERE tick = '%s'`
	sqlVars := make([]string, 0)
	for address, times := range mintTimes {
		sqlVars = append(sqlVars, fmt.Sprintf(sqlVar, address, times))
	}
	finalSql := sqlPrefix + "\n" + strings.Join(sqlVars, "\n") + fmt.Sprintf(sqlSuffix, tick)
	result := dbClient.Exec(finalSql)
	if result.Error != nil {
		xylog.Logger.Warnf("update mint times err %s", result.Error)
		return 0, result.Error
	}
	xylog.Logger.Debugf("update mint times affected %d", result.RowsAffected)
	return result.RowsAffected, nil
}

func (p *Protocol) insertMintTimes(tick string, mintTimes map[string]uint64) (int64, error) {
	if len(mintTimes) == 0 {
		return 0, nil
	}
	dbClient := p.cache.GetDBClient().SqlDB
	addressMintTimes := make([]*model.OpbrcAddressMintTimes, 0)
	for address, times := range mintTimes {
		addressMintTimes = append(addressMintTimes, &model.OpbrcAddressMintTimes{
			Chain:              chainName,
			Protocol:           protocolName,
			Address:            strings.ToLower(address),
			Tick:               strings.ToLower(tick),
			MintTimes:          times,
			CurrentSMMintTimes: 0,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		})
	}
	result := dbClient.Table(model.OpbrcAddressMintTimes{}.TableName()).Save(addressMintTimes)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (p *Protocol) updateInscriptionExtOnDeploy(block *xycommon.RpcBlock, tx *xycommon.RpcTransaction, deploy Deploy) (int64, error) {
	dbClient := p.cache.GetDBClient().SqlDB

	updateMap := map[string]interface{}{}
	updateMap["mspan"] = deploy.Mspan.IntPart()
	updateMap["cost"] = deploy.Cost.IntPart()
	updateMap["mcount"] = deploy.Mcount.IntPart()
	updateMap["sm"] = deploy.Sm.IntPart()
	updateMap["dr"] = deploy.Dr

	updateMap["settled_block_number"] = block.Number.Uint64()
	endBlockNumber := block.Number.Int64() + deploy.Mspan.IntPart()*60*60
	updateMap["end_block_number"] = endBlockNumber
	updateMap["start_block_number"] = block.Number.Uint64()
	avgSettleQty := deploy.Max.IntPart() / (deploy.Mspan.IntPart() * 60 / deploy.Sm.IntPart())
	updateMap["avg_settle_qty"] = avgSettleQty
	updateMap["updated_at"] = time.Now()
	updateMap["progress"] = 0

	result := dbClient.Table(model.OpbrcInscriptionExt{}.TableName()).Where("tick = ?", strings.ToLower(deploy.Tick)).UpdateColumns(updateMap)
	if result.Error != nil {
		xylog.Logger.Warnf("update inscription ext err %s", result.Error)
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func (p *Protocol) notSettledTick() []*model.OpbrcInscriptionExt {
	dbClient := p.cache.GetDBClient().SqlDB
	var insExt []*model.OpbrcInscriptionExt
	err := dbClient.Where("progress = ?", 0).Find(&insExt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || err != nil {
		return nil
	}
	return insExt
}

func (p *Protocol) insertListTx() {

}
