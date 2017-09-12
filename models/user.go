package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model

	Name         string `form:"name"`
	Wechat       string
	Gender       string
	Role         string
	Birthday     *time.Time
	Phone        string
	Seller       string `gorm:"index"`
	SourceWechat string
	Source       string
	Remark       string
	BuyTimes     uint
	LastBuyTime  *time.Time

	Addresses []Address
}

func (user User) DisplayName() string {
	return user.Name
}
