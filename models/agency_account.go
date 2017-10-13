package models

import (
	"github.com/jinzhu/gorm"
)

type AgencyAccount struct {
	gorm.Model
	
	AppId     string `gorm:"unique_index:uni_app_openid"`
	OpenId    string `gorm:"unique_index:uni_app_openid"`
	Name      string
	AvatarUrl string
	
	AgencyId uint
	Agency   Agency
}
