package utils

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/qor/l10n"
	"github.com/qor/qor/utils"
	"github.com/reechou/real-erp/auth"
	"github.com/reechou/real-erp/models"
)

// GetCurrentUser get current user from request
func GetCurrentUser(req *http.Request) *models.User {
	if currentUser, ok := auth.Auth.GetCurrentUser(req).(*models.User); ok {
		return currentUser
	}
	return nil
}

// GetCurrentLocale get current locale from request
func GetCurrentLocale(req *http.Request) string {
	locale := l10n.Global
	if cookie, err := req.Cookie("locale"); err == nil {
		locale = cookie.Value
	}
	return locale
}

// GetDB get DB from request
func GetDB(req *http.Request) *gorm.DB {
	if db := utils.GetDBFromRequest(req); db != nil {
		return db
	}
	return models.DB
}

// URLParam get url params from request
func URLParam(name string, req *http.Request) string {
	return chi.URLParam(req, name)
}

func init() {
	l10n.Global = "zh-CN"
}
