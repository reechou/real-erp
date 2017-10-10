package models

import (
	"fmt"
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qor/transition"
	"github.com/reechou/holmes"
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
	Seller             string  `gorm:"index"`
	transition.Transition
}

func (order *Order) Amount() (amount float32) {
	for _, orderItem := range order.OrderItems {
		amount += orderItem.Price
	}
	return
}

// 更新订单总金额
func (order *Order) BeforeCreate(tx *gorm.DB) (err error) {
	order.PaymentAmount = order.Amount()
	return
}

func (order *Order) BeforeUpdate(tx *gorm.DB) (err error) {
	oldOrder := new(Order)
	tx.First(oldOrder, order.ID)
	if oldOrder.TrackingNumber != "" {
		if oldOrder.State != order.State {
			// 更新状态
			return
		}
		return errors.New("订单状态不是未发货状态, 不能更新.")
	}
	if len(order.OrderItems) != 0 {
		order.PaymentAmount = order.Amount()
	}
	return
}

// 更新用户购买记录
func (order *Order) AfterCreate(tx *gorm.DB) (err error) {
	order.User.BuyTimes++
	err = tx.Model(&order.User).
		Select("buy_times", "last_buy_time").
		Updates(map[string]interface{}{"buy_times": order.User.BuyTimes, "last_buy_time": time.Now()}).Error
	if err != nil {
		holmes.Error("update user buy info error: %v", err)
		return
	}
	return
}

func (order *Order) BeforeDelete(tx *gorm.DB) (err error) {
	if order.TrackingNumber != "" {
		return errors.New("订单状态不是未发货状态, 不能删除.")
	}
	return
}

//func (order *Order) BeforeDelete(tx *gorm.DB) (err error) {
//	err = tx.Model(order).Related(&order.OrderItems).Error
//	if err != nil {
//		holmes.Error("before delete get order items error: %v", err)
//		return
//	}
//	for _, v := range order.OrderItems {
//		if v.ID == 0 {
//			continue
//		}
//		tx.Delete(&v)
//		// 返还库存
//		err = tx.Table("product_variations").
//			Where("id = ?", v.ProductVariationID).
//			UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", v.Quantity)).Error
//	}
//	return
//}

//func (orderItem *OrderItem) BeforeUpdate(tx *gorm.DB) (err error) {
//	oldOrderItem := new(OrderItem)
//	tx.First(oldOrderItem, orderItem.ID)
//	if orderItem.ProductVariation.ID != 0 {
//		rowsAffected := tx.Table("product_variations").
//			Where("id = ? AND available_quantity >= ?", orderItem.ProductVariation.ID, orderItem.Quantity).
//			UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", orderItem.Quantity)).RowsAffected
//		if rowsAffected == 0 {
//			err = errors.New("ProductVariation's AvailableQuantity < OrderItem's Quantity")
//			return
//		}
//		orderItem.ProductVariation.AvailableQuantity -= orderItem.Quantity
//
//		tx.Table("product_variations").
//			Where("id = ?", oldOrderItem.ProductVariationID).
//			UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", oldOrderItem.Quantity))
//	} else {
//		if orderItem.Quantity > oldOrderItem.Quantity {
//			addQuantity := orderItem.Quantity - oldOrderItem.Quantity
//			rowsAffected := tx.Table("product_variations").
//				Where("id = ? AND available_quantity >= ?", orderItem.ProductVariationID, addQuantity).
//				UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", addQuantity)).RowsAffected
//			if rowsAffected == 0 {
//				err = errors.New("ProductVariation's AvailableQuantity < OrderItem's Quantity")
//				return
//			}
//		} else if orderItem.Quantity < oldOrderItem.Quantity {
//			tx.Table("product_variations").
//				Where("id = ?", orderItem.ProductVariationID).
//				UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", oldOrderItem.Quantity-orderItem.Quantity))
//		}
//	}
//	return
//}

//func (orderItem *OrderItem) AfterCreate(tx *gorm.DB) (err error) {
//	holmes.Debug("order item: %+v", orderItem)
//	if orderItem.ProductVariationID == 0 {
//		return errors.New("Please select the product.")
//	}
//	if orderItem.Quantity == 0 {
//		return
//	}
//	// 减库存
//	rowsAffected := tx.Table("product_variations").
//		Where("id = ? AND available_quantity >= ?", orderItem.ProductVariationID, orderItem.Quantity).
//		UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", orderItem.Quantity)).RowsAffected
//	if rowsAffected == 0 {
//		err = errors.New(fmt.Sprintf("ProductVariation[%d]'s AvailableQuantity < OrderItem's Quantity[%d]", orderItem.ProductVariationID, orderItem.Quantity))
//		return
//	}
//	return
//}

var (
	OrderState = transition.New(&Order{})
	ItemState  = transition.New(&OrderItem{})
)

func init() {
	// Define Order's States
	OrderState.Initial("draft")
	OrderState.State("paid").Enter(func(value interface{}, tx *gorm.DB) (err error) {
		o := value.(*Order)
		for i := 0; i < len(o.OrderItems); i++ {
			if err = ItemState.Trigger("pay", &o.OrderItems[i], tx); err != nil {
				return err
			}
		}
		return nil
	})
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
			// 减库存
			rowsAffected := tx.Table("product_variations").
				Where("id = ? AND available_quantity >= ?", item.ProductVariationID, item.Quantity).
				UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", item.Quantity)).RowsAffected
			if rowsAffected == 0 {
				err = errors.New(fmt.Sprintf("ProductVariation[%d]'s AvailableQuantity < OrderItem's Quantity[%d]", item.ProductVariationID, item.Quantity))
				return
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
				// 返还库存
				if err = tx.Table("product_variations").
					Where("id = ?", item.ProductVariationID).
					UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", item.Quantity)).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	OrderState.Event("pay").To("paid").From("draft")
	OrderState.Event("ship").To("shipped").From("paid")
	OrderState.Event("complete").To("completed").From("shipped")
	OrderState.Event("return").To("returned").From("shipped", "completed")

	// Define ItemItem's States
	ItemState.Initial("draft")
	ItemState.State("paid")
	ItemState.State("shipped")
	ItemState.State("completed")
	ItemState.State("returned")

	ItemState.Event("pay").To("paid").From("draft")
	ItemState.Event("ship").To("shipped").From("paid")
	ItemState.Event("complete").To("completed").From("shipped")
	ItemState.Event("return").To("returned").From("shipped", "completed")
}
