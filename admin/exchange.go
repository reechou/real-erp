package admin

import (
	"github.com/qor/exchange"
	"github.com/qor/qor"
	"github.com/qor/qor/resource"
	"github.com/qor/qor/utils"
	"github.com/qor/validations"

	"github.com/reechou/real-erp/models"
)

var ProductExchange *exchange.Resource
var OrderExchange *exchange.Resource

func InitExchange() {
	ProductExchange = exchange.NewResource(&models.Product{}, exchange.Config{PrimaryField: "Code"})
	ProductExchange.Meta(&exchange.Meta{Name: "Code"})
	ProductExchange.Meta(&exchange.Meta{Name: "Name"})
	ProductExchange.Meta(&exchange.Meta{Name: "Price"})

	ProductExchange.AddValidator(func(record interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if utils.ToInt(metaValues.Get("Price").Value) < 0 {
			return validations.NewError(record, "Price", "price can't less than 0")
		}
		return nil
	})

	OrderExchange = exchange.NewResource(&models.Order{}, exchange.Config{PrimaryField: "ID"})
	OrderExchange.Meta(&exchange.Meta{Name: "ID", Header: "订单编号"})
	OrderExchange.Meta(&exchange.Meta{Name: "ShippingAddress.ContactName", Header: "收件人"})
	OrderExchange.Meta(&exchange.Meta{Name: "ShippingAddress.Phone", Header: "手机"})
	OrderExchange.Meta(&exchange.Meta{
		Name:   "ShippingAddress.AddressDetail",
		Header: "地址",
	})

	OrderExchange.AddValidator(func(record interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if utils.ToString(metaValues.Get("ShippingAddress.ContactName").Value) == "" {
			return validations.NewError(record, "ShippingAddress.ContactName", "ShippingAddress.ContactName can't be nil")
		}
		return nil
	})
}
