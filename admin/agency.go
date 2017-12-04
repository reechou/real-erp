package admin

import (
	"errors"
	//"bytes"
	//"html/template"
	//"fmt"

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
	// 代理
	agency := Admin.AddResource(&models.Agency{}, &admin.Config{Menu: []string{"代理管理"}, Name: "代理"})
	agency.Meta(&admin.Meta{Name: "Name", Label: "姓名"})
	agency.Meta(&admin.Meta{Name: "Phone", Label: "电话"})
	agency.Meta(&admin.Meta{Name: "IDCardNumber", Label: "身份证"})
	agency.Meta(&admin.Meta{Name: "Superior", Label: "上级"})
	agency.Meta(&admin.Meta{Name: "PurchaseTimes", Label: "进货次数", Permission: roles.Allow(roles.Read, roles.Anyone)})
	agency.Meta(&admin.Meta{Name: "LastPurchaseTime", Label: "最后进货时间", Permission: roles.Allow(roles.Read, roles.Anyone)})
	agency.Meta(&admin.Meta{Name: "Seller", Label: "销售员", Permission: roles.Allow(roles.Read, roles.Anyone)})
	agencyLevelsMeta := agency.Meta(&admin.Meta{
		Name:       "AgencyLevels",
		Label:      "代理等级汇总",
		Permission: roles.Allow(roles.Read, roles.Anyone),
	})
	agencyLevelsMeta.SetFormattedValuer(func(record interface{}, context *qor.Context) interface{} {
		variations := agencyLevelsMeta.GetValuer()(record, context).([]models.AgencyLevel)
		var results []string
		for _, v := range variations {
			results = append(results, v.AgencyLevelInfo())
		}
		return results
	})
	agencyLevelsMetaResource := agencyLevelsMeta.Resource
	agencyLevelsMetaResource.Meta(&admin.Meta{Name: "Category", Label: "分类"})
	alcMeta := agencyLevelsMetaResource.Meta(&admin.Meta{Name: "AgencyLevelConfig", Label: "代理等级"})
	alcMeta.SetFormattedValuer(func(record interface{}, context *qor.Context) interface{} {
		alc := alcMeta.GetValuer()(record, context).(models.AgencyLevelConfig)
		return alc.AgencyLevelConfigInfo()
	})
	agencyLevelsMetaResource.Meta(&admin.Meta{Name: "PurchaseCumulativeAmount", Label: "累计进货金额"})
	agencyProductQuantitiesResource := agencyLevelsMetaResource.Meta(&admin.Meta{
		Name:       "AgencyProductQuantities",
		Label:      "代理商品剩余量",
		Permission: roles.Allow(roles.Read, roles.Anyone),
	}).Resource
	apqProductVariationMeta := agencyProductQuantitiesResource.Meta(&admin.Meta{Name: "ProductVariation", Label: "商品"})
	apqProductVariationMeta.SetFormattedValuer(func(record interface{}, context *qor.Context) interface{} {
		pv := apqProductVariationMeta.GetValuer()(record, context).(models.ProductVariation)
		return pv.SKU
	})
	agencyProductQuantitiesResource.Meta(&admin.Meta{Name: "Quantity", Label: "剩余量"})
	
	subordinateAgencyMeta := agency.Meta(&admin.Meta{Name: "SubordinateAgency", Label: "下级代理列表", Permission: roles.Allow(roles.Read, roles.Anyone)})
	subordinateAgencyMeta.SetValuer(func(record interface{}, context *qor.Context) interface{} {
		if a, ok := record.(*models.Agency); ok {
			var agencies []models.Agency
			context.GetDB().Where("superior_id = ?", a.ID).Find(&agencies)
			return agencies
		}
		return nil
	})
	subordinateAgencyResource := subordinateAgencyMeta.Resource
	subordinateAgencyResource.Meta(&admin.Meta{Name: "Name", Label: "下级代理名字"})
	subordinateAgencyResource.EditAttrs("Name")

	agency.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if u, ok := value.(*models.Agency); ok {
			if u.ID == 0 {
				// create
				u.Seller = context.CurrentUser.DisplayName()
			}
		}
		return nil
	})

	agency.Scope(&admin.Scope{
		Default: true,
		Handler: func(db *gorm.DB, context *qor.Context) *gorm.DB {
			if IsAdmin(context) {
				return db
			}
			return db.Where("seller = ?", context.CurrentUser.DisplayName())
		},
	})
	
	agency.IndexAttrs("-SubordinateAgency")
	agency.ShowAttrs(
		&admin.Section{
			Title: "基本信息",
			Rows: [][]string{
				{"Name", "Phone"},
				{"IDCardNumber"},
				{"Superior", "PurchaseTimes", "LastPurchaseTime"},
				{"Seller"},
			},
		},
		"AgencyLevels",
		"SubordinateAgency",
	)
	agency.NewAttrs("-Superior")

	// 代理配置
	agencyLevelConfig := Admin.AddResource(&models.AgencyLevelConfig{}, &admin.Config{Menu: []string{"代理管理"}, Name: "配置"})
	agencyLevelConfig.Meta(&admin.Meta{Name: "Category", Label: "商品分类"})
	agencyLevelConfig.Meta(&admin.Meta{Name: "Level", Label: "等级"})
	agencyLevelConfig.Meta(&admin.Meta{Name: "CumulativeAmount", Label: "进货累积金额"})
	purchasePricesMeta := agencyLevelConfig.Meta(&admin.Meta{
		Name:  "PurchasePrices",
		Label: "商品进货价格",
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

	// 代理订单
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
	order.Meta(&admin.Meta{
		Name:  "State",
		Label: "订单状态",
		Valuer: func(record interface{}, ctx *qor.Context) interface{} {
			order := record.(*models.AgencyOrder)
			if state, ok := OrderStateMap[order.State]; ok {
				return state
			}
			return ""
		},
	})
	order.Meta(&admin.Meta{Name: "CreatedAt", Label: "创建时间"})
	order.Meta(&admin.Meta{Name: "Seller", Label: "销售员", Permission: roles.Allow(roles.Read, UserPermissions[USER_PERMISSION_ADMIN])})

	orderItemMeta := order.Meta(&admin.Meta{Name: "AgencyOrderItems", Label: "订单商品"})
	orderItemMetaResource := orderItemMeta.Resource
	orderItemMetaResource.Meta(&admin.Meta{
		Name:   "ProductVariation",
		Config: &admin.SelectOneConfig{Collection: productVariationCollection},
		Label:  "商品",
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
				oi.OrderStatus = 0
				oi.Seller = context.CurrentUser.DisplayName()
			}
		}
		return nil
	})

	order.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if o, ok := value.(*models.AgencyOrder); ok {
			if o.ID == 0 {
				// create
				o.OrderStatus = 0
				o.AgencySrc = 0
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
	// 更新代理金额和剩余库存
	//order.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
	//	if o, ok := value.(*models.AgencyOrder); ok {
	//		agencyId := o.AgencyID
	//		if agencyId == 0 {
	//			agencyId = o.Agency.ID
	//		}
	//		holmes.Debug("orderID: %v , agencyID: %v", o.ID, agencyId)
	//		//holmes.Debug(" ------- order processor: %+v", o)
	//		for i := 0; i < len(o.AgencyOrderItems); i++ {
	//			var diffPrice float32
	//			var diffQuantity int
	//			var productVariationID uint
	//			if o.AgencyOrderItems[i].ID != 0 {
	//				oldOrderItem := new(models.AgencyOrderItem)
	//				context.DB.First(oldOrderItem, o.AgencyOrderItems[i].ID)
	//				diffPrice = o.AgencyOrderItems[i].Price - oldOrderItem.Price
	//				diffQuantity = int(o.AgencyOrderItems[i].Quantity) - int(oldOrderItem.Quantity)
	//				productVariationID = o.AgencyOrderItems[i].ProductVariationID
	//				// order item update
	//				context.DB.First(&o.AgencyOrderItems[i].ProductVariation, o.AgencyOrderItems[i].ProductVariationID)
	//				context.DB.First(&o.AgencyOrderItems[i].ProductVariation.Product, o.AgencyOrderItems[i].ProductVariation.ProductID)
	//				holmes.Debug("[in order item update] order product categoryId: %v", o.AgencyOrderItems[i].ProductVariation.Product.CategoryId)
	//			} else {
	//				diffPrice = o.AgencyOrderItems[i].Price
	//				diffQuantity = int(o.AgencyOrderItems[i].Quantity)
	//				productVariationID = o.AgencyOrderItems[i].ProductVariation.ID
	//				// order item create
	//				context.DB.First(&o.AgencyOrderItems[i].ProductVariation.Product, o.AgencyOrderItems[i].ProductVariation.ProductID)
	//				holmes.Debug("[in order item create] order product categoryId: %v", o.AgencyOrderItems[i].ProductVariation.Product.CategoryId)
	//			}
	//			categoryId := o.AgencyOrderItems[i].ProductVariation.Product.CategoryId
	//			holmes.Debug("diff price: %v diff quantity: %v", diffPrice, diffQuantity)
	//			if diffPrice == 0 && diffQuantity == 0 {
	//				continue
	//			}
	//			agencyLevel := new(models.AgencyLevel)
	//			rowsAffected := context.DB.Where("agency_id = ? AND category_id = ?", agencyId, categoryId).First(agencyLevel).RowsAffected
	//			if rowsAffected == 0 {
	//				// get agency level config
	//				alc := new(models.AgencyLevelConfig)
	//				rowsAffected = context.DB.Where("category_id = ? AND cumulative_amount <= ?", categoryId, diffPrice).Order("cumulative_amount desc").First(alc).RowsAffected
	//				if rowsAffected != 0 {
	//					agencyLevel.AgencyLevelConfigID = alc.ID
	//				}
	//				agencyLevel.AgencyID = agencyId
	//				agencyLevel.CategoryID = categoryId
	//				agencyLevel.PurchaseCumulativeAmount = diffPrice
	//				// create agency level
	//				if err := context.DB.Save(agencyLevel).Error; err != nil {
	//					holmes.Error("save agency level error: %v", err)
	//					return err
	//				}
	//				holmes.Debug("agency level in create: %+v", agencyLevel)
	//				// create agency level product quantity
	//				apq := new(models.AgencyProductQuantity)
	//				apq.AgencyLevelID = agencyLevel.ID
	//				apq.ProductVariationID = productVariationID
	//				apq.Quantity = uint(diffQuantity)
	//				if err := context.DB.Save(apq).Error; err != nil {
	//					holmes.Error("save agency product quantity error: %v", err)
	//					return err
	//				}
	//				//holmes.Debug("agency product quantity in create: %+v", apq)
	//			} else {
	//				agencyLevel.PurchaseCumulativeAmount += diffPrice
	//				// get agency level config
	//				alc := new(models.AgencyLevelConfig)
	//				rowsAffected = context.DB.Where("category_id = ? AND cumulative_amount <= ?", categoryId, agencyLevel.PurchaseCumulativeAmount).Order("cumulative_amount desc").First(alc).RowsAffected
	//				if rowsAffected != 0 {
	//					if alc.ID != agencyLevel.AgencyLevelConfigID {
	//						holmes.Info("agency[%d] has chaged its agency level to id[%d]", agencyLevel.AgencyID, alc.ID)
	//						agencyLevel.AgencyLevelConfigID = alc.ID
	//					}
	//				}
	//				// save agency level
	//				if err := context.DB.Save(agencyLevel).Error; err != nil {
	//					holmes.Error("save agency level error: %v", err)
	//					return err
	//				}
	//				//holmes.Debug("agency level in save: %+v", agencyLevel)
	//				// get agency
	//				apq := new(models.AgencyProductQuantity)
	//				rowsAffected = context.DB.Where("agency_level_id = ? AND product_variation_id = ?", agencyLevel.ID, productVariationID).First(apq).RowsAffected
	//				if rowsAffected == 0 {
	//					apq.AgencyLevelID = agencyLevel.ID
	//					apq.ProductVariationID = productVariationID
	//				}
	//				apq.Quantity = uint(int(apq.Quantity) + diffQuantity)
	//				if err := context.DB.Save(apq).Error; err != nil {
	//					holmes.Error("save agency product quantity error: %v", err)
	//					return err
	//				}
	//				//holmes.Debug("agency product quantity in save: %+v", apq)
	//			}
	//		}
	//	}
	//	return nil
	//})

	order.Scope(&admin.Scope{
		Default: true,
		Handler: func(db *gorm.DB, context *qor.Context) *gorm.DB {
			if IsAdmin(context) {
				return db.Where("order_status = 0")
			}
			return db.Where("seller = ? AND order_status = 0 AND agency_src = 0", context.CurrentUser.DisplayName())
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
	order.NewAttrs("-AbandonedReason", "-PaymentAmount", "-CreatedAt", "-ShippedAt", "-CompletedAt", "-ReturnedAt", "-OrderStatus", "-AgencySrc")
	order.EditAttrs("-AbandonedReason", "-State", "-CreatedAt", "-ShippedAt", "-CompletedAt", "-ReturnedAt", "-OrderStatus", "-AgencySrc")
	order.SearchAttrs("User.Name", "User.Wechat", "ShippingAddress.ContactName", "ShippingAddress.AddressDetail")

	activity.Register(order)

	Admin.AddSearchResource(agency, order)
}
