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
	Seller           string

	AgencyLevels []AgencyLevel
	SubordinateAgency []Agency
}

type AgencyProductQuantity struct {
	gorm.Model
	AgencyLevelID uint `gorm:"unique_index:uni_agency_product_quantity"`
	
	ProductVariationID uint `gorm:"unique_index:uni_agency_product_quantity"`
	ProductVariation   ProductVariation
	Quantity           uint
}

type AgencyLevel struct {
	gorm.Model
	AgencyID uint `gorm:"unique_index:uni_agency_level"`

	CategoryID          uint `gorm:"unique_index:uni_agency_level"`
	Category            Category
	AgencyLevelConfigID uint
	AgencyLevelConfig   AgencyLevelConfig

	PurchaseCumulativeAmount float32                 // 该分类总进货金额
	AgencyProductQuantities  []AgencyProductQuantity // 剩余量
}

func (agencyLevel AgencyLevel) AgencyLevelInfo() string {
	DB.First(&agencyLevel.Category, agencyLevel.CategoryID)
	DB.First(&agencyLevel.AgencyLevelConfig, agencyLevel.AgencyLevelConfigID)
	return fmt.Sprintf("%s (等级: %d)", agencyLevel.Category.Name, agencyLevel.AgencyLevelConfig.Level)
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

	CategoryID uint `gorm:"unique_index:uni_agency_level_config"`
	Category   Category

	Level uint `gorm:"unique_index:uni_agency_level_config"`
	
	CumulativeAmount float32
	PurchasePrices   []AgencyPurchasePrice
}

func (agencyLevelConfig AgencyLevelConfig) AgencyLevelConfigInfo() string {
	return fmt.Sprintf("等级: %d", agencyLevelConfig.Level)
}
