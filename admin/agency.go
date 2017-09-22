package admin

import (
	"errors"

	"github.com/jinzhu/gorm"
	"github.com/qor/activity"
	"github.com/qor/admin"
	"github.com/qor/qor"
	"github.com/qor/qor/resource"
	"github.com/qor/roles"
	"github.com/qor/transition"
	"github.com/reechou/holmes"
	"github.com/reechou/real-erp/models"
)

func initAgencyAdmin() {
	agency := Admin.AddResource(&models.Agency{}, &admin.Config{Menu: []string{"代理管理"}, Name: "代理"})
	agency.Meta(&admin.Meta{Name: "Name", Label: "姓名"})
	agency.Meta(&admin.Meta{Name: "Phone", Label: "电话"})
	agency.Meta(&admin.Meta{Name: "IDCardNumber", Label: "身份证"})
	agency.Meta(&admin.Meta{Name: "Superior", Label: "上级"})

	agencyLevelConfig := Admin.AddResource(&models.AgencyLevelConfig{}, &admin.Config{Menu: []string{"代理管理"}, Name: "配置"})
	agencyLevelConfig.Meta(&admin.Meta{Name: "Category", Label: "商品分类"})
	agencyLevelConfig.Meta(&admin.Meta{Name: "PurchaseCumulativeAmount", Label: "进货累积金额"})
	purchasePricesMeta := agencyLevelConfig.Meta(&admin.Meta{
		Name:  "PurchasePrices",
		Label: "进货商品价格",
	})
	purchasePricesMeta.SetFormattedValuer(func(record interface{}, context *qor.Context) interface{} {
		variations := purchasePricesMeta.GetValuer()(record, context).([]models.AgencyPurchasePrice)
		var results []string
		for _, v := range variations {
			results = append(results, v.AgencyPurchasePriceInfo())
		}
		return results
	})
	purchasePricesResource := purchasePricesMeta.Resource
	purchasePricesResource.Meta(&admin.Meta{
		Name:   "ProductVariation",
		Config: &admin.SelectOneConfig{Collection: productVariationCollection},
		Label:  "进货商品",
	})
	purchasePricesResource.Meta(&admin.Meta{Name: "PurchasePrice", Label: "进货价格"})
	agencyLevelConfig.Filter(&admin.Filter{
		Name:   "Category",
		Config: &admin.SelectOneConfig{RemoteDataResource: Admin.GetResource("分类")},
		Label:  "代理商品分类",
	})

	order := Admin.AddResource(&models.AgencyOrder{}, &admin.Config{Menu: []string{"代理管理"}, Name: "进货订单"})
	shippingAddressMeta := order.Meta(&admin.Meta{Name: "ShippingAddress", Type: "single_edit", Label: "邮寄地址"})
	shippingAddressMetaResource := shippingAddressMeta.Resource
	shippingAddressMetaResource.Meta(&admin.Meta{Name: "ContactName", Label: "联系人"})
	shippingAddressMetaResource.Meta(&admin.Meta{Name: "Phone", Label: "电话"})
	shippingAddressMetaResource.Meta(&admin.Meta{Name: "Province", Label: "省份", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	shippingAddressMetaResource.Meta(&admin.Meta{Name: "City", Label: "城市", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	shippingAddressMetaResource.Meta(&admin.Meta{Name: "AddressDetail", Label: "详细地址"})
	shippingAddressMetaResource.EditAttrs(
		&admin.Section{
			Rows: [][]string{
				{"ContactName", "Phone"},
				{"Province", "City"},
				{"AddressDetail"},
			}},
	)
	shippingAddressMetaResource.NewAttrs(shippingAddressMetaResource.EditAttrs())
	shippingAddressMetaResource.ShowAttrs(shippingAddressMetaResource.EditAttrs())
	order.Meta(&admin.Meta{Name: "ShippedAt", Type: "date", Label: "发货时间"})
	order.Meta(&admin.Meta{Name: "CompletedAt", Type: "date", Label: "完成时间"})
	order.Meta(&admin.Meta{Name: "ReturnedAt", Type: "date", Label: "退货时间"})
	order.Meta(&admin.Meta{Name: "PaymentAmount", Label: "订单金额", Permission: roles.Allow(roles.Read, roles.Anyone)})
	order.Meta(&admin.Meta{Name: "Express", Label: "快递公司", Permission: roles.Allow(roles.Read, roles.Anyone)})
	order.Meta(&admin.Meta{Name: "TrackingNumber", Label: "快递单号", Permission: roles.Allow(roles.Read, roles.Anyone)})
	order.Meta(&admin.Meta{Name: "Agency", Label: "代理"})
	order.Meta(&admin.Meta{Name: "State", Label: "订单状态"})
	order.Meta(&admin.Meta{Name: "CreatedAt", Label: "创建时间"})
	order.Meta(&admin.Meta{Name: "Seller", Label: "销售员", Permission: roles.Allow(roles.Read, UserPermissions[USER_PERMISSION_ADMIN])})

	orderItemMeta := order.Meta(&admin.Meta{Name: "AgencyOrderItems", Label: "订单商品"})
	orderItemMetaResource := orderItemMeta.Resource
	orderItemMetaResource.Meta(&admin.Meta{
		Name:   "ProductVariation",
		Config: &admin.SelectOneConfig{Collection: productVariationCollection},
		Label:  "商品",
		Valuer: func(record interface{}, ctx *qor.Context) interface{} {
			orderItem := record.(*models.AgencyOrderItem)
			if orderItem.ProductVariationID != 0 {
				db := ctx.GetDB()
				db.First(&orderItem.ProductVariation, orderItem.ProductVariationID)
				return orderItem.ProductVariation.SKU
			}
			return ""
		},
	})
	orderItemMetaResource.Meta(&admin.Meta{Name: "Quantity", Label: "数量"})
	orderItemMetaResource.Meta(&admin.Meta{Name: "Price", Label: "总价"})
	orderItemMetaResource.Meta(&admin.Meta{Name: "State", Label: "订单状态"})
	orderItemMetaResource.EditAttrs(
		&admin.Section{
			Rows: [][]string{
				{"ProductVariation"},
				{"Quantity", "Price", "State"},
			}},
	)
	orderItemMetaResource.NewAttrs(orderItemMetaResource.EditAttrs())
	orderItemMetaResource.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if oi, ok := value.(*models.AgencyOrderItem); ok {
			if oi.ID == 0 {
				// create
				oi.Seller = context.CurrentUser.DisplayName()
			}
		}
		return nil
	})

	order.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if o, ok := value.(*models.AgencyOrder); ok {
			if o.ID == 0 {
				// create
				o.Seller = context.CurrentUser.DisplayName()
				models.AgencyOrderState.Trigger("pay", o, context.DB)
			} else {
				for i := 0; i < len(o.AgencyOrderItems); i++ {
					if o.AgencyOrderItems[i].ID == 0 {
						o.AgencyOrderItems[i].State = o.State
					}
				}
			}
		}
		return nil
	})

	order.Scope(&admin.Scope{
		Default: true,
		Handler: func(db *gorm.DB, context *qor.Context) *gorm.DB {
			if IsAdmin(context) {
				return db
			}
			return db.Where("seller = ?", context.CurrentUser.DisplayName())
		},
	})

	for i, state := range OrderState {
		var state = state
		order.Scope(&admin.Scope{
			Name:  state,
			Label: OrderStateLabel[i],
			Group: "订单状态",
			Handler: func(db *gorm.DB, context *qor.Context) *gorm.DB {
				return db.Where(models.AgencyOrder{Transition: transition.Transition{State: state}})
			},
		})
	}

	type trackingNumberArgument struct {
		Express        string
		TrackingNumber string
	}

	trackingResource := Admin.NewResource(&trackingNumberArgument{}, &admin.Config{Name: "快递单号"})
	trackingResource.Meta(&admin.Meta{Name: "Express", Config: &admin.SelectOneConfig{Collection: Expresses}, Label: "快递公司"})
	trackingResource.Meta(&admin.Meta{Name: "TrackingNumber", Label: "快递单号"})

	order.Action(&admin.Action{
		Name: "发货",
		Handler: func(argument *admin.ActionArgument) error {
			var (
				tx                     = argument.Context.GetDB().Begin()
				trackingNumberArgument = argument.Argument.(*trackingNumberArgument)
			)
			holmes.Debug("agency order tracking number: %v", trackingNumberArgument)
			if trackingNumberArgument.TrackingNumber != "" {
				for _, record := range argument.FindSelectedRecords() {
					order := record.(*models.AgencyOrder)
					order.TrackingNumber = trackingNumberArgument.TrackingNumber
					order.Express = trackingNumberArgument.Express
					if err := models.AgencyOrderState.Trigger("ship", order, tx, "tracking number "+trackingNumberArgument.TrackingNumber); err != nil {
						holmes.Error("agency order[%d] trigger [ship] error: %v", order.ID, err)
						return err
					}
					if err := tx.Save(order).Error; err != nil {
						tx.Rollback()
						return err
					}
				}
			} else {
				return errors.New("invalid shipment number")
			}
			tx.Commit()
			return nil
		},
		Visible: func(record interface{}, context *admin.Context) bool {
			if order, ok := record.(*models.AgencyOrder); ok {
				return order.State == "paid"
			}
			return false
		},
		Resource: trackingResource,
		Modes:    []string{"show", "edit", "menu_item"},
	})
	order.Action(&admin.Action{
		Name: "订单完成",
		Handler: func(argument *admin.ActionArgument) error {
			for _, order := range argument.FindSelectedRecords() {
				db := argument.Context.GetDB()
				if err := models.OrderState.Trigger("complete", order.(*models.AgencyOrder), db); err != nil {
					return err
				}
				db.Select("state").Save(order)
			}
			return nil
		},
		Visible: func(record interface{}, context *admin.Context) bool {
			if order, ok := record.(*models.AgencyOrder); ok {
				return order.State == "shipped"
			}
			return false
		},
		Modes: []string{"show", "edit", "menu_item"},
	})
	order.Action(&admin.Action{
		Name: "退货",
		Handler: func(argument *admin.ActionArgument) error {
			for _, order := range argument.FindSelectedRecords() {
				db := argument.Context.GetDB()
				if err := models.OrderState.Trigger("return", order.(*models.AgencyOrder), db); err != nil {
					return err
				}
				db.Select("state").Save(order)
			}
			return nil
		},
		Visible: func(record interface{}, context *admin.Context) bool {
			if order, ok := record.(*models.AgencyOrder); ok {
				return order.State == "completed" || order.State == "shipped"
			}
			return false
		},
		Modes: []string{"show", "edit", "menu_item"},
	})

	order.IndexAttrs("User", "PaymentAmount", "ShippedAt", "State", "ShippingAddress", "CreatedAt", "Seller")
	order.NewAttrs("-AbandonedReason", "-PaymentAmount", "-CreatedAt", "-ShippedAt", "-CompletedAt", "-ReturnedAt")
	order.EditAttrs("-AbandonedReason", "-State", "-CreatedAt", "-ShippedAt", "-CompletedAt", "-ReturnedAt")
	order.SearchAttrs("User.Name", "User.Wechat", "ShippingAddress.ContactName", "ShippingAddress.AddressDetail")

	activity.Register(order)

	Admin.AddSearchResource(agency, order)
}
