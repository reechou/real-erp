package admin

import (
	"bytes"
	"errors"
	"html/template"
	"strconv"

	"github.com/jinzhu/gorm"
	"github.com/qor/activity"
	"github.com/qor/admin"
	//"github.com/qor/i18n/exchange_actions"
	"github.com/qor/media"
	"github.com/qor/media/media_library"
	"github.com/qor/qor"
	"github.com/qor/qor/resource"
	"github.com/qor/roles"
	"github.com/qor/transition"
	"github.com/reechou/holmes"
	"github.com/reechou/real-erp/auth"
	"github.com/reechou/real-erp/i18n"
	"github.com/reechou/real-erp/models"
)

var Admin *admin.Admin
var Genders = []string{"男", "女", "不知"}
var Expresses = []string{"顺丰", "中通", "圆通", "EMS"}
var OrderState = []string{"paid", "shipped", "completed", "returned"}
var OrderStateLabel = []string{"待发货", "已发货", "已完成", "已退货"}
var UserRoles = []string{"Admin", "Maintainer", "Member"}
var UserPermissions = []string{"admin", "maintainer", "member"}

const (
	USER_PERMISSION_ADMIN = iota
	USER_PERMISSION_MAINTAINER
	USER_PERMISSION_MEMBER
)

func InitAdmin() {
	Admin = admin.New(&qor.Config{DB: models.DB})
	Admin.SetSiteName("REAL ERP")
	Admin.SetAuth(auth.AdminAuth{})

	Admin.AddMenu(&admin.Menu{Name: "数据概览", Link: "/admin"})

	Admin.AddMenu(&admin.Menu{Name: "产品管理"})

	category := Admin.AddResource(&models.Category{}, &admin.Config{Menu: []string{"产品管理"}, Name: "分类", Priority: -3})
	category.Meta(&admin.Meta{Name: "Name", Label: "分类名"})
	category.Meta(&admin.Meta{Name: "Code", Label: "代码"})

	// Add ProductImage as Media Library
	productImagesResource := Admin.AddResource(&models.ProductImage{}, &admin.Config{
		Menu:       []string{"产品管理"},
		Name:       "图片",
		Priority:   -2,
		Permission: roles.Allow(roles.CRUD, UserPermissions[USER_PERMISSION_ADMIN]),
	})
	productImagesResource.Meta(&admin.Meta{Name: "Title", Label: "图片名"})
	productImagesResource.Meta(&admin.Meta{Name: "Category", Label: "所属分类"})
	productImagesResource.Filter(&admin.Filter{
		Name:       "SelectedType",
		Label:      "资源类型",
		Operations: []string{"contains"},
		Config:     &admin.SelectOneConfig{Collection: [][]string{{"video", "视频"}, {"image", "图片"}, {"file", "文件"}, {"video_link", "视频链接"}}},
	})
	productImagesResource.Filter(&admin.Filter{
		Name:   "Category",
		Config: &admin.SelectOneConfig{RemoteDataResource: category},
		Label:  "分类",
	})
	productImagesResource.IndexAttrs("File", "Title")

	productPurchaseResource := Admin.AddResource(&models.ProductPurchase{}, &admin.Config{
		Menu:       []string{"产品管理"},
		Name:       "进货记录",
		Priority:   -1,
		Permission: roles.Allow(roles.CRUD, UserPermissions[USER_PERMISSION_ADMIN]),
	})
	productPurchaseResource.Meta(&admin.Meta{Name: "Quantity", Label: "进货数量"})
	productPurchaseResource.Meta(&admin.Meta{Name: "CreatedAt", Label: "进货时间"})
	productPurchaseResource.Meta(&admin.Meta{
		Name:   "ProductVariation",
		Config: &admin.SelectOneConfig{Collection: productVariationCollection},
		Label:  "商品",
		Valuer: func(record interface{}, ctx *qor.Context) interface{} {
			productPurchase := record.(*models.ProductPurchase)
			if productPurchase.ProductVariationID != 0 {
				db := ctx.GetDB()
				db.First(&productPurchase.ProductVariation, productPurchase.ProductVariationID)
				return productPurchase.ProductVariation.SKU
			}
			return ""
		},
	})
	productPurchaseResource.Filter(&admin.Filter{
		Name:   "ProductVariation",
		Config: &admin.SelectOneConfig{Collection: productVariationCollection},
		Label:  "商品SKU",
	})

	product := Admin.AddResource(&models.Product{}, &admin.Config{
		Menu:       []string{"产品管理"},
		Name:       "产品",
		Permission: roles.Deny(roles.Update, UserPermissions[USER_PERMISSION_MAINTAINER], UserPermissions[USER_PERMISSION_MEMBER]).Deny(roles.Create, UserPermissions[USER_PERMISSION_MAINTAINER], UserPermissions[USER_PERMISSION_MEMBER]),
	})
	product.Meta(&admin.Meta{Name: "Category", Config: &admin.SelectOneConfig{AllowBlank: true}})
	product.Meta(&admin.Meta{Name: "MainImage", Config: &media_library.MediaBoxConfig{
		RemoteDataResource: productImagesResource,
		Max:                1,
		Sizes: map[string]*media.Size{
			"main": {Width: 560, Height: 700},
		},
	}, Label: "主图"})
	product.Meta(&admin.Meta{Name: "MainImageURL", Valuer: func(record interface{}, context *qor.Context) interface{} {
		if p, ok := record.(*models.Product); ok {
			result := bytes.NewBufferString("")
			tmpl, _ := template.New("").Parse("<img src='{{.image}}'></img>")
			tmpl.Execute(result, map[string]string{"image": p.MainImageURL()})
			return template.HTML(result.String())
		}
		return ""
	}})
	product.UseTheme("grid")
	product.Meta(&admin.Meta{Name: "Name", Label: "名字"})
	product.Meta(&admin.Meta{Name: "Code", Label: "代码"})
	product.Meta(&admin.Meta{Name: "Price", Label: "价格", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	product.Meta(&admin.Meta{Name: "Description", Label: "商品详情"})
	product.Meta(&admin.Meta{Name: "Category", Label: "商品分类"})
	product.Meta(&admin.Meta{Name: "Variations", Label: "商品sku"})

	product.SearchAttrs("Name", "Code", "Category.Name")
	product.IndexAttrs("MainImageURL", "Name", "Price", "Variations")
	product.EditAttrs(
		&admin.Section{
			Title: "基本信息",
			Rows: [][]string{
				{"Name"},
				{"Code", "Price"},
				{"MainImage"},
			}},
		&admin.Section{
			Title: "分类",
			Rows: [][]string{
				{"Category"},
			}},
		"Description",
		"Variations",
	)
	product.NewAttrs(product.EditAttrs())

	variationsMeta := product.Meta(&admin.Meta{Name: "Variations"})
	variationsMeta.SetFormattedValuer(func(record interface{}, context *qor.Context) interface{} {
		variations := variationsMeta.GetValuer()(record, context).([]models.ProductVariation)
		var results []string
		for _, v := range variations {
			results = append(results, v.ProductVariationInfo())
		}
		return results
	})
	variationsResource := variationsMeta.Resource
	variationsResource.Permission = roles.Deny(roles.Update, UserPermissions[USER_PERMISSION_MAINTAINER], UserPermissions[USER_PERMISSION_MEMBER]).Deny(roles.Create, UserPermissions[USER_PERMISSION_MAINTAINER], UserPermissions[USER_PERMISSION_MEMBER])
	variationsResource.NewAttrs("-Product")
	variationsResource.EditAttrs("-ID", "-Product")
	variationsResource.Meta(&admin.Meta{Name: "Price", Label: "价格"})
	variationsResource.Meta(&admin.Meta{Name: "AvailableQuantity", Label: "库存"})

	user := Admin.AddResource(&models.User{}, &admin.Config{Menu: []string{"用户管理"}, Name: "用户"})
	user.Meta(&admin.Meta{Name: "Gender", Config: &admin.SelectOneConfig{Collection: Genders}, Label: "性别"})
	user.Meta(&admin.Meta{Name: "Birthday", Type: "date", Label: "生日"})
	user.Meta(&admin.Meta{
		Name:       "Role",
		Config:     &admin.SelectOneConfig{Collection: UserRoles},
		Label:      "权限",
		Permission: roles.Allow(roles.Read, UserPermissions[USER_PERMISSION_ADMIN]).Allow(roles.Update, UserPermissions[USER_PERMISSION_ADMIN]),
	})
	user.Meta(&admin.Meta{Name: "Name", Label: "用户名"})
	user.Meta(&admin.Meta{Name: "Wechat", Label: "用户微信"})
	user.Meta(&admin.Meta{Name: "Seller", Label: "销售员", Permission: roles.Allow(roles.CRUD, UserPermissions[USER_PERMISSION_ADMIN])})
	user.Meta(&admin.Meta{Name: "SourceWechat", Label: "来源微信"})
	user.Meta(&admin.Meta{Name: "Source", Label: "用户来源"})
	user.Meta(&admin.Meta{Name: "Remark", Label: "备注"})
	user.Meta(&admin.Meta{Name: "BuyTimes", Label: "购买次数"})
	user.Meta(&admin.Meta{Name: "LastBuyTime", Label: "最后购买时间"})
	user.Meta(&admin.Meta{Name: "Addresses", Label: "地址列表"})
	user.Meta(&admin.Meta{Name: "CreatedAt", Label: "创建时间"})
	user.Meta(&admin.Meta{Name: "Phone", Label: "电话"})
	user.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if u, ok := value.(*models.User); ok {
			if u.ID == 0 {
				// create
				u.Role = UserRoles[USER_PERMISSION_MEMBER]
				u.Seller = context.CurrentUser.DisplayName()
			}
		}
		return nil
	})

	user.Scope(&admin.Scope{
		Default: true,
		Handler: func(db *gorm.DB, context *qor.Context) *gorm.DB {
			if IsAdmin(context) {
				return db
			}
			return db.Where("seller = ?", context.CurrentUser.DisplayName())
		},
	})

	user.IndexAttrs("ID", "Name", "Wechat", "Gender", "Phone", "BuyTimes", "LastBuyTime", "SourceWechat", "CreatedAt", "Remark")
	user.EditAttrs(
		&admin.Section{
			Title: "基本信息",
			Rows: [][]string{
				{"Name", "Wechat", "Gender"},
				{"Birthday", "Role", "Phone"},
				{"Seller", "SourceWechat"},
				{"Source", "Remark"},
			},
		},
		"Addresses",
	)
	user.NewAttrs(user.EditAttrs())
	//user.ShowAttrs(user.EditAttrs())

	addressesResource := user.Meta(&admin.Meta{Name: "Addresses"}).Resource
	addressesResource.Meta(&admin.Meta{Name: "ContactName", Label: "联系人名字"})
	addressesResource.Meta(&admin.Meta{Name: "Phone", Label: "联系电话"})
	addressesResource.Meta(&admin.Meta{Name: "Province", Label: "省份", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	addressesResource.Meta(&admin.Meta{Name: "City", Label: "城市", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	addressesResource.Meta(&admin.Meta{Name: "AddressDetail", Label: "详细地址"})
	addressesResource.EditAttrs(
		&admin.Section{
			Title: "地址",
			Rows: [][]string{
				{"ContactName", "Phone"},
				{"Province", "City"},
				{"AddressDetail"},
			}},
	)
	addressesResource.NewAttrs(addressesResource.EditAttrs())

	order := Admin.AddResource(&models.Order{}, &admin.Config{Menu: []string{"订单管理"}, Name: "订单"})
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
	order.Meta(&admin.Meta{Name: "PaymentAmount", Label: "订单金额"})
	order.Meta(&admin.Meta{Name: "Express", Label: "快递公司"})
	order.Meta(&admin.Meta{Name: "TrackingNumber", Label: "快递单号"})
	order.Meta(&admin.Meta{Name: "User", Label: "用户"})
	order.Meta(&admin.Meta{Name: "State", Label: "订单状态"})
	order.Meta(&admin.Meta{Name: "CreatedAt", Label: "创建时间"})
	order.Meta(&admin.Meta{Name: "Seller", Label: "销售员", Permission: roles.Allow(roles.CRUD, UserPermissions[USER_PERMISSION_ADMIN])})

	orderItemMeta := order.Meta(&admin.Meta{Name: "OrderItems", Label: "订单商品"})
	orderItemMetaResource := orderItemMeta.Resource
	orderItemMetaResource.Meta(&admin.Meta{
		Name:   "ProductVariation",
		Config: &admin.SelectOneConfig{Collection: productVariationCollection},
		Label:  "商品",
		Valuer: func(record interface{}, ctx *qor.Context) interface{} {
			orderItem := record.(*models.OrderItem)
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

	order.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if o, ok := value.(*models.Order); ok {
			if o.ID == 0 {
				// create
				o.Seller = context.CurrentUser.DisplayName()
				for i := 0; i < len(o.OrderItems); i++ {
					o.OrderItems[i].Seller = context.CurrentUser.DisplayName()
				}
				models.OrderState.Trigger("pay", o, context.DB)
				//holmes.Debug("order processor: %+v", o)
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
				return db.Where(models.Order{Transition: transition.Transition{State: state}})
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
			holmes.Debug("tracking number: %v", trackingNumberArgument)
			if trackingNumberArgument.TrackingNumber != "" {
				for _, record := range argument.FindSelectedRecords() {
					order := record.(*models.Order)
					order.TrackingNumber = trackingNumberArgument.TrackingNumber
					order.Express = trackingNumberArgument.Express
					if err := models.OrderState.Trigger("ship", order, tx, "tracking number "+trackingNumberArgument.TrackingNumber); err != nil {
						holmes.Error("order[%d] trigger [ship] error: %v", order.ID, err)
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
			if order, ok := record.(*models.Order); ok {
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
				if err := models.OrderState.Trigger("complete", order.(*models.Order), db); err != nil {
					return err
				}
				db.Select("state").Save(order)
			}
			return nil
		},
		Visible: func(record interface{}, context *admin.Context) bool {
			if order, ok := record.(*models.Order); ok {
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
				if err := models.OrderState.Trigger("return", order.(*models.Order), db); err != nil {
					return err
				}
				db.Select("state").Save(order)
			}
			return nil
		},
		Visible: func(record interface{}, context *admin.Context) bool {
			if order, ok := record.(*models.Order); ok {
				return order.State == "completed" || order.State == "shipped"
			}
			return false
		},
		Modes: []string{"show", "edit", "menu_item"},
	})

	order.IndexAttrs("User", "PaymentAmount", "ShippedAt", "State", "ShippingAddress", "CreatedAt", "Seller")
	order.NewAttrs("-AbandonedReason", "-PaymentAmount", "-CreatedAt", "-ShippedAt", "-CompletedAt", "-ReturnedAt")
	order.EditAttrs("-AbandonedReason", "-State", "-CreatedAt", "-ShippedAt", "-CompletedAt", "-ReturnedAt")
	order.ShowAttrs("-AbandonedReason", "-State", "-CreatedAt")
	order.SearchAttrs("User.Name", "User.Wechat", "ShippingAddress.ContactName", "ShippingAddress.AddressDetail")

	activity.Register(order)

	Admin.AddSearchResource(product, user, order)

	Admin.AddResource(i18n.I18n, &admin.Config{Menu: []string{"附加工具"}, Priority: 1, Invisible: true})

	Worker := getWorker()
	//exchange_actions.RegisterExchangeJobs(i18n.I18n, Worker)
	Admin.AddResource(Worker, &admin.Config{Menu: []string{"附加工具"}, Name: "任务"})

	initFuncMap()
	initReport()
}

func productVariationCollection(resource interface{}, context *qor.Context) (results [][]string) {
	for _, productVariation := range models.ProductVariations() {
		results = append(results, []string{strconv.Itoa(int(productVariation.ID)), productVariation.Stringify()})
	}
	return
}
