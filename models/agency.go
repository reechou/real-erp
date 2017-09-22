package models

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

type Agency struct {
	gorm.Model

	Name             string `form:"name"`
	Phone            string
	IDCardNumber     string
	SuperiorID       uint
	Superior         *Agency
	PurchaseTimes    uint
	LastPurchaseTime *time.Time

	AgencyLevels []AgencyLevel
}

type AgencyLevel struct {
	gorm.Model
	AgencyID uint

	CategoryID          uint
	Category            Category
	AgencyLevelConfigID uint
	AgencyLevelConfig   AgencyLevelConfig

	PurchaseCumulativeAmount float32
	Quantity                 uint
}

type AgencyPurchasePrice struct {
	gorm.Model
	AgencyLevelConfigID uint

	ProductVariationID uint
	ProductVariation   ProductVariation
	PurchasePrice      float32
}

func (agencyPurchasePrice AgencyPurchasePrice) AgencyPurchasePriceInfo() string {
	DB.First(&agencyPurchasePrice.ProductVariation, agencyPurchasePrice.ProductVariationID)
	return fmt.Sprintf("%s (￥%.2f)", agencyPurchasePrice.ProductVariation.SKU, agencyPurchasePrice.PurchasePrice)
}

type AgencyLevelConfig struct {
	gorm.Model

	CategoryID uint
	Category   Category

	Level uint

	PurchaseCumulativeAmount float32
	PurchasePrices           []AgencyPurchasePrice
}
