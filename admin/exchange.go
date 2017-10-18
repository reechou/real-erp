package admin

import (
	"fmt"
	
	"github.com/qor/exchange"
	"github.com/qor/qor"
	"github.com/qor/qor/resource"
	"github.com/qor/validations"

	"github.com/reechou/real-erp/models"
	"github.com/reechou/holmes"
)

var OrderExchange *exchange.Resource
var OrderExchangeImport *exchange.Resource

func InitExchange() {
	OrderExchange = exchange.NewResource(&models.Order{}, exchange.Config{PrimaryField: "ID"})
	OrderExchange.Meta(&exchange.Meta{Name: "ID", Header: "订单编号"})
	OrderExchange.Meta(&exchange.Meta{Name: "ShippingAddress.ContactName", Header: "收件人"})
	OrderExchange.Meta(&exchange.Meta{Name: "ShippingAddress.Phone", Header: "手机"})
	OrderExchange.Meta(&exchange.Meta{
		Name:   "ShippingAddress.AddressDetail",
		Header: "地址",
	})
	OrderExchange.AddValidator(func(record interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		o := record.(*models.Order)
		if o.ID == 0 {
			return validations.NewError(record, "ID", fmt.Sprintf("未找到该订单编号: %v", record))
		}
		return nil
	})
	
	OrderExchangeImport = exchange.NewResource(&models.Order{}, exchange.Config{PrimaryField: "ID"})
	OrderExchangeImport.Meta(&exchange.Meta{Name: "ID", Header: "订单编号"})
	OrderExchangeImport.Meta(&exchange.Meta{Name: "ShippingAddress.ContactName", Header: "收件人"})
	OrderExchangeImport.Meta(&exchange.Meta{Name: "ShippingAddress.Phone", Header: "电话"})
	OrderExchangeImport.Meta(&exchange.Meta{Name: "Express", Header: "快递名称"})
	OrderExchangeImport.Meta(&exchange.Meta{Name: "TrackingNumber", Header: "快递单号"})
	OrderExchangeImport.Meta(&exchange.Meta{
		Name:   "ShippingAddress.AddressDetail",
		Header: "地址",
	})
	OrderExchangeImport.AddValidator(func(record interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		o := record.(*models.Order)
		if o.ID == 0 {
			return validations.NewError(record, "ID", fmt.Sprintf("未找到该订单编号"))
		}
		return nil
	})
	OrderExchangeImport.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if o, ok := value.(*models.Order); ok {
			if o.ID != 0 {
				// update
				// trigger ship
				if err := models.OrderState.Trigger("ship", o, context.DB, "tracking number "+o.TrackingNumber); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
