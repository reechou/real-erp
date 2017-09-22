package models

import (
	"fmt"
	"errors"
	"time"
	
	"github.com/jinzhu/gorm"
	"github.com/qor/transition"
	"github.com/reechou/holmes"
)

type AgencyOrder struct {
	gorm.Model
	AgencyID          uint
	Agency            Agency
	PaymentAmount     float32
	AbandonedReason   string
	Express           string
	TrackingNumber    string
	ShippedAt         *time.Time
	CompletedAt       *time.Time
	ReturnedAt        *time.Time
	ShippingAddressID uint `form:"shippingaddress"`
	ShippingAddress   Address
	AgencyOrderItems  []AgencyOrderItem
	Seller            string `gorm:"index"`
	transition.Transition
}

type AgencyOrderItem struct {
	gorm.Model
	AgencyOrderID      uint
	ProductVariationID uint `cartitem:"ProductVariationID"`
	ProductVariation   ProductVariation
	Quantity           uint    `cartitem:"Quantity"`
	Price              float32 // 总价
	Seller             string  `gorm:"index"`
	transition.Transition
}

func (order *AgencyOrder) Amount() (amount float32) {
	for _, orderItem := range order.AgencyOrderItems {
		amount += orderItem.Price
	}
	return
}

func (order *AgencyOrder) BeforeCreate(tx *gorm.DB) (err error) {
	order.PaymentAmount = order.Amount()
	return
}

func (order *AgencyOrder) BeforeUpdate(tx *gorm.DB) (err error) {
	if len(order.AgencyOrderItems) != 0 {
		order.PaymentAmount = order.Amount()
	}
	
	return
}

func (order *AgencyOrder) AfterCreate(tx *gorm.DB) (err error) {
	// 更新代理进货记录
	order.Agency.PurchaseTimes++
	err = tx.Model(&order.Agency).
		Select("purchase_times", "last_purchase_time").
		Updates(map[string]interface{}{"purchase_times": order.Agency.PurchaseTimes, "last_purchase_time": time.Now()}).Error
	if err != nil {
		holmes.Error("update agency purchase info error: %v", err)
		return
	}
	return
}

func (order *AgencyOrder) BeforeDelete(tx *gorm.DB) (err error) {
	err = tx.Model(order).Related(&order.AgencyOrderItems).Error
	if err != nil {
		holmes.Error("before delete get order items error: %v", err)
		return
	}
	for _, v := range order.AgencyOrderItems {
		if v.ID == 0 {
			continue
		}
		tx.Delete(&v)
		// 返还库存
		err = tx.Table("product_variations").
			Where("id = ?", v.ProductVariationID).
			UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", v.Quantity)).Error
	}
	return
}

func (orderItem *AgencyOrderItem) BeforeUpdate(tx *gorm.DB) (err error) {
	oldOrderItem := new(AgencyOrderItem)
	tx.First(oldOrderItem, orderItem.ID)
	if orderItem.ProductVariation.ID != 0 {
		rowsAffected := tx.Table("product_variations").
			Where("id = ? AND available_quantity >= ?", orderItem.ProductVariation.ID, orderItem.Quantity).
			UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", orderItem.Quantity)).RowsAffected
		if rowsAffected == 0 {
			err = errors.New("ProductVariation's AvailableQuantity < OrderItem's Quantity")
			return
		}
		orderItem.ProductVariation.AvailableQuantity -= orderItem.Quantity
		
		tx.Table("product_variations").
			Where("id = ?", oldOrderItem.ProductVariationID).
			UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", oldOrderItem.Quantity))
	} else {
		if orderItem.Quantity > oldOrderItem.Quantity {
			addQuantity := orderItem.Quantity - oldOrderItem.Quantity
			rowsAffected := tx.Table("product_variations").
				Where("id = ? AND available_quantity >= ?", orderItem.ProductVariationID, addQuantity).
				UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", addQuantity)).RowsAffected
			if rowsAffected == 0 {
				err = errors.New("ProductVariation's AvailableQuantity < OrderItem's Quantity")
				return
			}
		} else if orderItem.Quantity < oldOrderItem.Quantity {
			tx.Table("product_variations").
				Where("id = ?", orderItem.ProductVariationID).
				UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", oldOrderItem.Quantity-orderItem.Quantity))
		}
	}
	return
}

func (orderItem *AgencyOrderItem) AfterCreate(tx *gorm.DB) (err error) {
	if orderItem.ProductVariationID == 0 {
		return errors.New("Please select the product.")
	}
	if orderItem.Quantity == 0 {
		return
	}
	// 减库存
	rowsAffected := tx.Table("product_variations").
		Where("id = ? AND available_quantity >= ?", orderItem.ProductVariationID, orderItem.Quantity).
		UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", orderItem.Quantity)).RowsAffected
	if rowsAffected == 0 {
		err = errors.New(fmt.Sprintf("ProductVariation[%d]'s AvailableQuantity < OrderItem's Quantity[%d]", orderItem.ProductVariationID, orderItem.Quantity))
		return
	}
	return
}

var (
	AgencyOrderState = transition.New(&AgencyOrder{})
	AgencyItemState  = transition.New(&AgencyOrderItem{})
)

func init() {
	// Define Order's States
	AgencyOrderState.Initial("draft")
	AgencyOrderState.State("paid").Enter(func(value interface{}, tx *gorm.DB) (err error) {
		o := value.(*AgencyOrder)
		for i := 0; i < len(o.AgencyOrderItems); i++ {
			if err = AgencyItemState.Trigger("pay", &o.AgencyOrderItems[i], tx); err != nil {
				return err
			}
		}
		return nil
	})
	AgencyOrderState.State("shipped").Enter(func(value interface{}, tx *gorm.DB) (err error) {
		tx.Model(value).UpdateColumn("shipped_at", time.Now())
		
		var orderItems []AgencyOrderItem
		tx.Model(value).Association("AgencyOrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err = AgencyItemState.Trigger("ship", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	AgencyOrderState.State("completed").Enter(func(value interface{}, tx *gorm.DB) error {
		tx.Model(value).UpdateColumn("completed_at", time.Now())
		
		var orderItems []AgencyOrderItem
		tx.Model(value).Association("AgencyOrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err := AgencyItemState.Trigger("complete", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	AgencyOrderState.State("returned").Enter(func(value interface{}, tx *gorm.DB) error {
		tx.Model(value).UpdateColumn("returned_at", time.Now())
		
		var orderItems []AgencyOrderItem
		tx.Model(value).Association("AgencyOrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err := AgencyItemState.Trigger("return", &item, tx); err == nil {
				if err = tx.Select("state").Save(&item).Error; err != nil {
					return err
				}
				// 退货返还库存
				if err = tx.Table("product_variations").
					Where("id = ?", item.ProductVariationID).
					UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", item.Quantity)).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	
	AgencyOrderState.Event("pay").To("paid").From("draft")
	AgencyOrderState.Event("ship").To("shipped").From("paid")
	AgencyOrderState.Event("complete").To("completed").From("shipped")
	AgencyOrderState.Event("return").To("returned").From("shipped", "completed")
	
	// Define ItemItem's States
	AgencyItemState.Initial("draft")
	AgencyItemState.State("paid")
	AgencyItemState.State("shipped")
	AgencyItemState.State("completed")
	AgencyItemState.State("returned")
	
	AgencyItemState.Event("pay").To("paid").From("draft")
	AgencyItemState.Event("ship").To("shipped").From("paid")
	AgencyItemState.Event("complete").To("completed").From("shipped")
	AgencyItemState.Event("return").To("returned").From("shipped", "completed")
}