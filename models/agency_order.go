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

// 更新订单总金额
func (order *AgencyOrder) BeforeCreate(tx *gorm.DB) (err error) {
	order.PaymentAmount = order.Amount()
	return
}

func (order *AgencyOrder) BeforeUpdate(tx *gorm.DB) (err error) {
	oldOrder := new(AgencyOrder)
	tx.First(oldOrder, order.ID)
	if oldOrder.TrackingNumber != "" {
		if oldOrder.State != order.State {
			// 更新状态
			return
		}
		return errors.New("订单状态不是未发货状态, 不能更新.")
	}
	if len(order.AgencyOrderItems) != 0 {
		order.PaymentAmount = order.Amount()
	}
	return
}

// 更新代理进货记录
func (order *AgencyOrder) AfterCreate(tx *gorm.DB) (err error) {
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
	if order.TrackingNumber != "" {
		return errors.New("订单状态不是未发货状态, 不能删除.")
	}
	return
}

//func (order *AgencyOrder) BeforeDelete(tx *gorm.DB) (err error) {
//	holmes.Debug("agency order delete: %v", order)
//	err = tx.Model(order).Related(&order.AgencyOrderItems).Error
//	if err != nil {
//		holmes.Error("before delete get order items error: %v", err)
//		return
//	}
//	for _, v := range order.AgencyOrderItems {
//		if v.ID == 0 {
//			continue
//		}
//		tx.Delete(&v)
//		// 返还库存
//		//err = tx.Table("product_variations").
//		//	Where("id = ?", v.ProductVariationID).
//		//	UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", v.Quantity)).Error
//	}
//	return
//}
//
//// about order item
//func (orderItem *AgencyOrderItem) BeforeDelete(tx *gorm.DB) (err error) {
//	holmes.Debug("agency order item delete: %v", orderItem)
//	// 返还库存
//	err = tx.Table("product_variations").
//		Where("id = ?", orderItem.ProductVariationID).
//		UpdateColumn("available_quantity", gorm.Expr("available_quantity + ?", orderItem.Quantity)).Error
//	if err != nil {
//		holmes.Error("delete agency order item return quantity error: %v", err)
//		return
//	}
//	// 更新代理余额和代理库存
//	tx.First(&orderItem.ProductVariation, orderItem.ProductVariationID)
//	tx.First(&orderItem.ProductVariation.Product, orderItem.ProductVariation.ProductID)
//	order := new(AgencyOrder)
//	tx.First(order, orderItem.AgencyOrderID)
//	agencyLevel := new(AgencyLevel)
//	rowsAffected := tx.Where("agency_id = ? AND category_id = ?", order.AgencyID, orderItem.ProductVariation.Product.CategoryId).First(agencyLevel).RowsAffected
//	if rowsAffected != 0 {
//		if agencyLevel.PurchaseCumulativeAmount < orderItem.Price {
//			holmes.Error("agency level[%v] < order item price[%v]", agencyLevel, orderItem.Price)
//			return
//		}
//		agencyLevel.PurchaseCumulativeAmount -= orderItem.Price
//		alc := new(AgencyLevelConfig)
//		rowsAffected = tx.Where("category_id = ? AND cumulative_amount <= ?",
//			orderItem.ProductVariation.Product.CategoryId, agencyLevel.PurchaseCumulativeAmount).
//			Order("cumulative_amount desc").First(alc).RowsAffected
//		if rowsAffected != 0 {
//			if alc.ID != agencyLevel.AgencyLevelConfigID {
//				holmes.Info("[delte order item] agency[%d] has chaged its agency level to id[%d]", agencyLevel.AgencyID, alc.ID)
//				agencyLevel.AgencyLevelConfigID = alc.ID
//			}
//		}
//		if err = tx.Save(agencyLevel).Error; err != nil {
//			holmes.Error("save agency level error: %v", err)
//			return
//		}
//		apq := new(AgencyProductQuantity)
//		rowsAffected = tx.Where("agency_level_id = ? AND product_variation_id = ?",
//			agencyLevel.ID, orderItem.ProductVariationID).First(apq).RowsAffected
//		if rowsAffected != 0 {
//			if apq.Quantity < orderItem.Quantity {
//				holmes.Error("agency product quantity[%v] < order item quantity[%v]", apq, orderItem.Quantity)
//				return
//			}
//			apq.Quantity -= orderItem.Quantity
//			if err = tx.Save(apq).Error; err != nil {
//				holmes.Error("save agency product quantity error: %v", err)
//				return
//			}
//		}
//	}
//	return
//}
//
//func (orderItem *AgencyOrderItem) BeforeUpdate(tx *gorm.DB) (err error) {
//	oldOrderItem := new(AgencyOrderItem)
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
//
//func (orderItem *AgencyOrderItem) AfterCreate(tx *gorm.DB) (err error) {
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
		
		order, ok := value.(*AgencyOrder)
		if !ok {
			return errors.New("value is not agency order.")
		}
		var orderItems []AgencyOrderItem
		tx.Model(value).Association("AgencyOrderItems").Find(&orderItems)
		for _, item := range orderItems {
			if err = AgencyItemState.Trigger("ship", &item, tx); err == nil {
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
			// 更新代理金额和代理剩余库存
			tx.First(&item.ProductVariation, item.ProductVariationID)
			tx.First(&item.ProductVariation.Product, item.ProductVariation.ProductID)
			categoryId := item.ProductVariation.Product.CategoryId
			agencyLevel := new(AgencyLevel)
			rowsAffected = tx.Where("agency_id = ? AND category_id = ?", order.AgencyID, categoryId).First(agencyLevel).RowsAffected
			if rowsAffected == 0 {
				agencyLevel.PurchaseCumulativeAmount = item.Price
				// get agency level config
				alc := new(AgencyLevelConfig)
				rowsAffected = tx.Where("category_id = ? AND cumulative_amount <= ?", categoryId, agencyLevel.PurchaseCumulativeAmount).Order("cumulative_amount desc").First(alc).RowsAffected
				if rowsAffected != 0 {
					agencyLevel.AgencyLevelConfigID = alc.ID
				}
				agencyLevel.AgencyID = order.AgencyID
				agencyLevel.CategoryID = categoryId
				// create agency level
				if err := tx.Save(agencyLevel).Error; err != nil {
					holmes.Error("save agency level error: %v", err)
					return err
				}
				holmes.Debug("agency level in create: %+v", agencyLevel)
				// create agency level product quantity
				apq := new(AgencyProductQuantity)
				apq.AgencyLevelID = agencyLevel.ID
				apq.ProductVariationID = item.ProductVariationID
				apq.Quantity = item.Quantity
				if err := tx.Save(apq).Error; err != nil {
					holmes.Error("save agency product quantity error: %v", err)
					return err
				}
			} else {
				agencyLevel.PurchaseCumulativeAmount += item.Price
				// get agency level config
				alc := new(AgencyLevelConfig)
				rowsAffected = tx.Where("category_id = ? AND cumulative_amount <= ?", categoryId, agencyLevel.PurchaseCumulativeAmount).Order("cumulative_amount desc").First(alc).RowsAffected
				if rowsAffected != 0 {
					if alc.ID != agencyLevel.AgencyLevelConfigID {
						holmes.Info("agency[%d] has chaged its agency level to id[%d]", agencyLevel.AgencyID, alc.ID)
						agencyLevel.AgencyLevelConfigID = alc.ID
					}
				}
				// save agency level
				if err := tx.Save(agencyLevel).Error; err != nil {
					holmes.Error("save agency level error: %v", err)
					return err
				}
				// get agency
				apq := new(AgencyProductQuantity)
				rowsAffected = tx.Where("agency_level_id = ? AND product_variation_id = ?", agencyLevel.ID, item.ProductVariationID).First(apq).RowsAffected
				if rowsAffected == 0 {
					apq.AgencyLevelID = agencyLevel.ID
					apq.ProductVariationID = item.ProductVariationID
				}
				apq.Quantity += item.Quantity
				if err := tx.Save(apq).Error; err != nil {
					holmes.Error("save agency product quantity error: %v", err)
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