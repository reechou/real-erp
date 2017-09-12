package admin

import (
	"fmt"
	"path"
	"time"
	
	"github.com/jinzhu/gorm"
	"github.com/qor/exchange"
	"github.com/qor/exchange/backends/csv"
	"github.com/qor/qor"
	"github.com/qor/worker"
	"github.com/qor/admin"
	"github.com/qor/roles"
	"github.com/qor/qor/resource"

	"github.com/reechou/real-erp/models"
)

func getWorker() *worker.Worker {
	Worker := worker.New()
	
	type ExportOrdersArgument struct {
		CurrentUser string
		IfAdmin     bool
		StartTime   *time.Time
		EndTime     *time.Time
	}
	ordersArgumentResource := Admin.NewResource(&ExportOrdersArgument{})
	ordersArgumentResource.Meta(&admin.Meta{Name: "StartTime", Label: "开始时间"})
	ordersArgumentResource.Meta(&admin.Meta{Name: "EndTime", Label: "结束时间"})
	ordersArgumentResource.Meta(&admin.Meta{Name: "CurrentUser", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	ordersArgumentResource.Meta(&admin.Meta{Name: "IfAdmin", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	ordersArgumentResource.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if u, ok := value.(*ExportOrdersArgument); ok {
			u.CurrentUser = context.CurrentUser.DisplayName()
			if CurrentUserIfAdmin(context) {
				u.IfAdmin = true
			}
		}
		return nil
	})

	Worker.RegisterJob(&worker.Job{
		Name:  "导出订单",
		Group: "订单管理",
		Handler: func(arg interface{}, qorJob worker.QorJobInterface) error {
			qorJob.AddLog("导出订单中...")
			orderArg := arg.(*ExportOrdersArgument)
			qorJob.AddLog(fmt.Sprintf("订单查询: %+v", orderArg))
			
			var db *gorm.DB
			var fileName string
			if orderArg.IfAdmin {
				db = models.DB.Where("created_at > ? AND created_at < ?", orderArg.StartTime, orderArg.EndTime).Preload("ShippingAddress")
				fileName = fmt.Sprintf("/downloads/admin_order.%v.csv", time.Now().Format("2006-01-02_15:04:05.999999"))
			} else {
				db = models.DB.Where("created_at > ? AND created_at < ? AND seller = ?", orderArg.StartTime, orderArg.EndTime, orderArg.CurrentUser).Preload("ShippingAddress")
				fileName = fmt.Sprintf("/downloads/%v_order.%v.csv", orderArg.CurrentUser, time.Now().Format("2006-01-02_15:04:05.999999"))
			}
			context := &qor.Context{DB: db}
			if err := OrderExchange.Export(
				csv.New(path.Join("public", fileName)),
				context,
				func(progress exchange.Progress) error {
					o := progress.Value.(*models.Order)
					qorJob.AddLog(fmt.Sprintf("%v/%v 正在导出订单, 邮寄信息: [%v - %v - %v]",
						progress.Current, progress.Total, o.ShippingAddress.ContactName, o.ShippingAddress.Phone, o.ShippingAddress.AddressDetail))
					return nil
				},
			); err != nil {
				qorJob.AddLog(err.Error())
			}
			qorJob.SetProgressText(fmt.Sprintf("<a href='%v'>下载订单</a>", fileName))
			return nil
		},
		Resource: ordersArgumentResource,
	})

	return Worker
}
