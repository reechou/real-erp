package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

type Address struct {
	gorm.Model
	UserID uint

	ContactName   string `form:"contact-name"`
	Phone         string `form:"phone"`
	Province      string `form:"province"`
	City          string `form:"city"`
	AddressDetail string `form:"address-detail"`
}

func (address Address) Stringify() string {
	if address.City == "" && address.Province == "" {
		return address.AddressDetail
	}
	return fmt.Sprintf("%v, %v, %v", address.AddressDetail, address.City, address.Province)
}
