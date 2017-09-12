package models

import (
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qor/transition"
)

type Order struct {
	gorm.Model
	UserID            uint
	User              User
	PaymentAmount     float32
	AbandonedReason   string
	Express           string
	TrackingNumber    string
	ShippedAt         *time.Time
	CompletedAt       *time.Time
	ReturnedAt        *time.Time
	ShippingAddressID uint `form:"shippingaddress"`
	ShippingAddress   Address
	OrderItems        []OrderItem
	Seller            string `gorm:"index"`
	transition.Transition
}

type OrderItem struct {
	gorm.Model
	OrderID            uint
	ProductVariationID uint `cartitem:"ProductVariationID"`
	ProductVariation   ProductVariation
	Quantity           uint    `cartitem:"Quantity"`
	Price              float32 // 总价
	transition.Transition
}

func (order *Order) Amount() (amount float32) {
	for _, orderItem := range order.OrderItems {
		amount += orderItem.Price
	}
	return
}

func (order *Order) BeforeCreate(tx *gorm.DB) (err error) {
	order.PaymentAmount = order.Amount()
	return
}

func (order *Order) AfterCreate(tx *gorm.DB) (err error) {
	// 更新用户购买记录
	order.User.BuyTimes++
	err = tx.Model(&order.User).
		Select("buy_times", "last_buy_time").
		Updates(map[string]interface{}{"buy_times": order.User.BuyTimes, "last_buy_time": time.Now()}).Error
	return
}

func (orderItem *OrderItem) AfterCreate(tx *gorm.DB) (err error) {
	if orderItem.ProductVariation.AvailableQuantity < orderItem.Quantity {
		return errors.New("ProductVariation's AvailableQuantity < OrderItem's Quantity")
	}
	// 减库存
	orderItem.ProductVariation.AvailableQuantity -= orderItem.Quantity
	err = tx.Select("available_quantity").Save(&orderItem.ProductVariation).Error
	return
}

var (
	OrderState = transition.New(&Order{})
	ItemState  = transition.New(&OrderItem{})
)

func init() {
	// Define Order's States
	OrderState.Initial("paid")
	OrderState.State("shipped").Enter(func(value interface{}, tx *gorm.DB) (err error) {
		tx.Model(value).UpdateColumn("shipped_at", time.Now())

		var orderItems []OrderItem
		tx.Model(value).Association("OrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err = ItemState.Trigger("ship", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	OrderState.State("completed").Enter(func(value interface{}, tx *gorm.DB) error {
		tx.Model(value).UpdateColumn("completed_at", time.Now())

		var orderItems []OrderItem
		tx.Model(value).Association("OrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err := ItemState.Trigger("complete", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	OrderState.State("returned").Enter(func(value interface{}, tx *gorm.DB) error {
		tx.Model(value).UpdateColumn("returned_at", time.Now())

		var orderItems []OrderItem
		tx.Model(value).Association("OrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err := ItemState.Trigger("return", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	OrderState.Event("ship").To("shipped").From("paid")
	OrderState.Event("complete").To("completed").From("shipped")
	OrderState.Event("return").To("returned").From("shipped", "completed")

	// Define ItemItem's States
	ItemState.Initial("paid")
	ItemState.State("shipped")
	ItemState.State("completed")
	ItemState.State("returned")

	ItemState.Event("ship").To("shipped").From("paid")
	ItemState.Event("complete").To("completed").From("shipped")
	ItemState.Event("return").To("returned").From("shipped", "completed")
}
