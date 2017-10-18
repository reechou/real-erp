package admin

import (
	"fmt"
	"path"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qor/admin"
	"github.com/qor/exchange"
	"github.com/qor/exchange/backends/csv"
	"github.com/qor/qor"
	"github.com/qor/qor/resource"
	"github.com/qor/roles"
	"github.com/qor/worker"
	"github.com/qor/media/oss"

	"github.com/reechou/real-erp/models"
)

var WorkerOrderState = []string{"all", "paid", "shipped", "completed", "returned"}

func getWorker() *worker.Worker {
	Worker := worker.New()

	type ExportOrdersArgument struct {
		CurrentUser string
		IfAdmin     bool
		State       string
		StartTime   *time.Time
		EndTime     *time.Time
	}
	ordersArgumentResource := Admin.NewResource(&ExportOrdersArgument{})
	ordersArgumentResource.Meta(&admin.Meta{Name: "State", Config: &admin.SelectOneConfig{Collection: WorkerOrderState}, Label: "订单状态"})
	ordersArgumentResource.Meta(&admin.Meta{Name: "StartTime", Label: "开始时间"})
	ordersArgumentResource.Meta(&admin.Meta{Name: "EndTime", Label: "结束时间"})
	ordersArgumentResource.Meta(&admin.Meta{Name: "CurrentUser", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	ordersArgumentResource.Meta(&admin.Meta{Name: "IfAdmin", Permission: roles.Deny(roles.CRUD, roles.Anyone)})
	ordersArgumentResource.AddProcessor(func(value interface{}, metaValues *resource.MetaValues, context *qor.Context) error {
		if u, ok := value.(*ExportOrdersArgument); ok {
			u.CurrentUser = context.CurrentUser.DisplayName()
			if IsAdmin(context) {
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
				if orderArg.State == "all" {
					db = models.DB.Where("created_at > ? AND created_at < ?", orderArg.StartTime, orderArg.EndTime).Preload("ShippingAddress")
					fileName = fmt.Sprintf("/downloads/admin_order.%v.csv", time.Now().Format("2006-01-02_15:04:05.999999"))
				} else {
					db = models.DB.Where("created_at > ? AND created_at < ? AND state = ?", orderArg.StartTime, orderArg.EndTime, orderArg.State).Preload("ShippingAddress")
					fileName = fmt.Sprintf("/downloads/admin_order.%v.csv", time.Now().Format("2006-01-02_15:04:05.999999"))
				}
			} else {
				if orderArg.State == "all" {
					db = models.DB.Where("created_at > ? AND created_at < ? AND seller = ? AND state = ?",
						orderArg.StartTime, orderArg.EndTime, orderArg.CurrentUser, orderArg.State).Preload("ShippingAddress")
					fileName = fmt.Sprintf("/downloads/%v_order.%v.csv", orderArg.CurrentUser, time.Now().Format("2006-01-02_15:04:05.999999"))
				} else {
					db = models.DB.Where("created_at > ? AND created_at < ? AND seller = ?",
						orderArg.StartTime, orderArg.EndTime, orderArg.CurrentUser).Preload("ShippingAddress")
					fileName = fmt.Sprintf("/downloads/%v_order.%v.csv", orderArg.CurrentUser, time.Now().Format("2006-01-02_15:04:05.999999"))
				}
			}
			context := &qor.Context{DB: db}
			if err := OrderExchange.Export(
				csv.New(path.Join("public", fileName)),
				context,
				func(progress exchange.Progress) error {
					o := progress.Value.(*models.Order)
					qorJob.AddLog(fmt.Sprintf("%v/%v 正在导出订单, 邮寄信息: [%v - %v - %v]",
						progress.Current, progress.Total, o.ShippingAddress.ContactName, o.ShippingAddress.Phone, o.ShippingAddress.AddressDetail))
					if progress.Current == progress.Total {
						qorJob.AddLog("导出订单已完成")
					}
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
	
	type ImportOrdersArgument struct {
		File oss.OSS
	}
	
	Worker.RegisterJob(&worker.Job{
		Name:  "导入订单",
		Group: "订单管理",
		Handler: func(arg interface{}, qorJob worker.QorJobInterface) error {
			qorJob.AddLog("导入订单 快递单号 中...")
			orderArg := arg.(*ImportOrdersArgument)
			qorJob.AddLog(fmt.Sprintf("import orders file: %v", orderArg.File.URL()))
			
			context := &qor.Context{DB: models.DB}
			var errorCount uint
			if err := OrderExchangeImport.Import(
				csv.New(path.Join("public", orderArg.File.URL())),
				context,
				func(progress exchange.Progress) error {
					var cells = []worker.TableCell{
						{Value: fmt.Sprint(progress.Current)},
					}
					
					var hasError bool
					for _, cell := range progress.Cells {
						var tableCell = worker.TableCell{
							Value: fmt.Sprint(cell.Value),
						}
						
						if cell.Error != nil {
							hasError = true
							errorCount++
							tableCell.Error = cell.Error.Error()
						}
						
						cells = append(cells, tableCell)
					}
					
					if hasError {
						if errorCount == 1 {
							var headerCells = []worker.TableCell{
								{Value: "Line No."},
							}
							for _, cell := range progress.Cells {
								headerCells = append(headerCells, worker.TableCell{
									Value: cell.Header,
								})
							}
							qorJob.AddResultsRow(headerCells...)
						}
						
						qorJob.AddResultsRow(cells...)
					}
					
					o := progress.Value.(*models.Order)
					
					qorJob.SetProgress(uint(float32(progress.Current) / float32(progress.Total) * 100))
					qorJob.AddLog(fmt.Sprintf("\t -- %v", o))
					qorJob.AddLog(fmt.Sprintf("%d/%d 导入订单: %v %v %v", progress.Current, progress.Total, o.ID, o.Express, o.TrackingNumber))
					if progress.Current == progress.Total {
						qorJob.AddLog("导入订单已完成")
					}
					return nil
				},
			); err != nil {
				qorJob.AddLog(err.Error())
			}
			return nil
		},
		Resource: Admin.NewResource(&ImportOrdersArgument{}),
	})
	
	return Worker
}
