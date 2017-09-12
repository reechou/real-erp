package controllers

import (
	"html/template"
	"net/http"

	"github.com/qor/i18n/inline_edit"
	"github.com/qor/middlewares"
	"github.com/qor/render"
	"github.com/reechou/holmes"
	"github.com/reechou/real-erp/admin"
	"github.com/reechou/real-erp/auth"
	"github.com/reechou/real-erp/config"
	"github.com/reechou/real-erp/i18n"
	"github.com/reechou/real-erp/models"
	"github.com/reechou/real-erp/routes"
	"github.com/reechou/real-erp/utils"
)

type Logic struct {
	cfg *config.Config
}

func NewLogic(cfg *config.Config) *Logic {
	l := &Logic{
		cfg: cfg,
	}

	models.InitDB(cfg)

	i18n.InitI18n()

	auth.InitAuth()
	auth.InitAdminAuth()

	admin.InitFilebox()
	admin.InitExchange()

	admin.InitAdmin()

	return l
}

func (self *Logic) Run() {
	defer holmes.Start(holmes.LogFilePath("./log"),
		holmes.EveryDay,
		holmes.AlsoStdout,
		holmes.DebugLevel).Stop()

	mux := http.NewServeMux()
	mux.Handle("/", routes.Router())
	admin.Admin.MountTo("/admin", mux)
	admin.Filebox.MountTo("/downloads", mux)

	config.View.FuncMapMaker = func(render *render.Render, req *http.Request, w http.ResponseWriter) template.FuncMap {
		funcMap := template.FuncMap{}

		// Add `t` method
		for key, fc := range inline_edit.FuncMap(i18n.I18n, utils.GetCurrentLocale(req), false) {
			funcMap[key] = fc
		}

		funcMap["current_user"] = func() *models.User {
			return utils.GetCurrentUser(req)
		}

		return funcMap
	}

	holmes.Info("server listening on[%s]..", self.cfg.Host)
	if err := http.ListenAndServe(self.cfg.Host, middlewares.Apply(mux)); err != nil {
		panic(err)
	}
}
