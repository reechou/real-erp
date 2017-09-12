package admin

import (
	"github.com/qor/filebox"
	"github.com/qor/roles"

	"github.com/reechou/real-erp/auth"
	"github.com/reechou/real-erp/config"
)

var Filebox *filebox.Filebox

func InitFilebox() {
	Filebox = filebox.New(config.Root + "/public/downloads")
	Filebox.SetAuth(auth.AdminAuth{})
	dir := Filebox.AccessDir("/")
	dir.SetPermission(roles.Allow(roles.Read, "admin"))
}
