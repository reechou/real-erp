package auth

import (
	"net/http"

	"github.com/qor/admin"
	"github.com/qor/qor"
	"github.com/qor/roles"
	"github.com/reechou/real-erp/models"
)

func InitAdminAuth() {
	roles.Register("admin", func(req *http.Request, currentUser interface{}) bool {
		return currentUser != nil && currentUser.(*models.User).Role == "Admin"
	})

	roles.Register("maintainer", func(req *http.Request, currentUser interface{}) bool {
		return currentUser != nil && currentUser.(*models.User).Role == "Maintainer"
	})

	roles.Register("member", func(req *http.Request, currentUser interface{}) bool {
		return currentUser != nil && currentUser.(*models.User).Role == "Member"
	})
}

type AdminAuth struct {
}

func (AdminAuth) LoginURL(c *admin.Context) string {
	return "/auth/login"
}

func (AdminAuth) LogoutURL(c *admin.Context) string {
	return "/auth/logout"
}

func (AdminAuth) GetCurrentUser(c *admin.Context) qor.CurrentUser {
	currentUser, _ := Auth.GetCurrentUser(c.Request).(qor.CurrentUser)
	return currentUser
}
