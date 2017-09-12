package admin

import (
	"encoding/json"

	"github.com/qor/admin"

	"github.com/reechou/real-erp/models"
)

type Charts struct {
	Orders []models.Chart
	Users  []models.Chart
}

func ReportsDataHandler(context *admin.Context) {
	charts := &Charts{}
	startDate := context.Request.URL.Query().Get("startDate")
	endDate := context.Request.URL.Query().Get("endDate")

	if CurrentUserIfAdmin(context.Context) {
		charts.Orders = models.GetChartData("orders", startDate, endDate, "")
		charts.Users = models.GetChartData("users", startDate, endDate, "")
	} else {
		charts.Orders = models.GetChartData("orders", startDate, endDate, context.CurrentUser.DisplayName())
		charts.Users = models.GetChartData("users", startDate, endDate, context.CurrentUser.DisplayName())
	}

	b, _ := json.Marshal(charts)
	context.Writer.Write(b)
	return
}

func initReport() {
	Admin.GetRouter().Get("/reports", ReportsDataHandler)
}
