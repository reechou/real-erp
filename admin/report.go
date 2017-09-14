package admin

import (
	"encoding/json"

	"github.com/qor/admin"

	"github.com/reechou/real-erp/models"
)

type Charts struct {
	Orders            []models.Chart
	Users             []models.Chart
	Quantity          []models.Chart
	Amount            []models.Chart
	SellerPerformance []models.UserChart
}

func ReportsDataHandler(context *admin.Context) {
	charts := &Charts{}
	startDate := context.Request.URL.Query().Get("startDate")
	endDate := context.Request.URL.Query().Get("endDate")

	if IsAdmin(context.Context) {
		charts.Orders = models.GetChartData("orders", startDate, endDate, "")
		charts.Users = models.GetChartData("users", startDate, endDate, "")
		charts.Quantity = models.GetChartDataOfSum("order_items", "quantity", startDate, endDate, "")
		charts.Amount = models.GetChartDataOfSum("order_items", "price", startDate, endDate, "")
	} else {
		charts.Orders = models.GetChartData("orders", startDate, endDate, context.CurrentUser.DisplayName())
		charts.Users = models.GetChartData("users", startDate, endDate, context.CurrentUser.DisplayName())
		charts.Quantity = models.GetChartDataOfSum("order_items", "quantity", startDate, endDate, context.CurrentUser.DisplayName())
		charts.Amount = models.GetChartDataOfSum("order_items", "price", startDate, endDate, context.CurrentUser.DisplayName())
	}
	charts.SellerPerformance = models.GetUserChartDataOfSum("order_items", "price", startDate, endDate)

	b, _ := json.Marshal(charts)
	context.Writer.Write(b)
	return
}

func initReport() {
	Admin.GetRouter().Get("/reports", ReportsDataHandler)
}
