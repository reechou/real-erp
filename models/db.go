package models

import (
	"errors"
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/qor/activity"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/media"
	"github.com/qor/media/asset_manager"
	"github.com/qor/transition"
	"github.com/qor/validations"

	"github.com/reechou/real-erp/config"
)

var (
	DB *gorm.DB
)

func InitDB(cfg *config.Config) {
	var err error

	if cfg.DBInfo.Adapter == "mysql" {
		DB, err = gorm.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v)/%v?charset=utf8&parseTime=True&loc=Local", cfg.DBInfo.User, cfg.DBInfo.Pass, cfg.DBInfo.Host, cfg.DBInfo.DBName))
	} else if cfg.DBInfo.Adapter == "postgres" {
		DB, err = gorm.Open("postgres", fmt.Sprintf("postgres://%v:%v@%v/%v?sslmode=disable", cfg.DBInfo.User, cfg.DBInfo.Pass, cfg.DBInfo.Host, cfg.DBInfo.DBName))
	} else if cfg.DBInfo.Adapter == "sqlite" {
		DB, err = gorm.Open("sqlite3", fmt.Sprintf("%v/%v", os.TempDir(), cfg.DBInfo.DBName))
	} else {
		panic(errors.New("not supported database adapter"))
	}

	if err == nil {
		if os.Getenv("DEBUG") != "" {
			DB.LogMode(true)
		}
		//DB.LogMode(true)

		validations.RegisterCallbacks(DB)
		media.RegisterCallbacks(DB)
	} else {
		panic(err)
	}

	AutoMigrate(&asset_manager.AssetManager{})

	AutoMigrate(&Product{}, &ProductVariation{}, &ProductImage{}, &ProductPurchase{})
	AutoMigrate(&Category{})

	AutoMigrate(&Address{})
	AutoMigrate(&User{})
	AutoMigrate(&Order{}, &OrderItem{})
	AutoMigrate(&transition.StateChangeLog{})
	
	AutoMigrate(&Agency{}, &AgencyProductQuantity{}, &AgencyLevel{}, &AgencyLevelConfig{}, &AgencyPurchasePrice{}, &AgencyAccount{})
	AutoMigrate(&AgencyOrder{}, &AgencyOrderItem{})

	AutoMigrate(&activity.QorActivity{})

	AutoMigrate(&MediaLibrary{})

	AutoMigrate(&auth_identity.AuthIdentity{})
}

func AutoMigrate(values ...interface{}) {
	for _, value := range values {
		DB.AutoMigrate(value)
	}
}
