package auth

import (
	"time"

	"github.com/qor/auth"
	"github.com/qor/auth/authority"
	"github.com/reechou/real-erp/auth_themes/clean"
	"github.com/reechou/real-erp/config"
	"github.com/reechou/real-erp/models"
)

var (
	// Auth initialize Auth for Authentication
	Auth *auth.Auth

	// Authority initialize Authority for Authorization
	Authority *authority.Authority
)

func InitAuth() {
	Auth = clean.New(&auth.Config{
		DB:         models.DB,
		Render:     config.View,
		UserModel:  models.User{},
		Redirector: auth.Redirector{RedirectBack: config.RedirectBack},
	})

	// Authority initialize Authority for Authorization
	Authority = authority.New(&authority.Config{
		Auth: Auth,
	})

	Authority.Register("logged_in_half_hour", authority.Rule{TimeoutSinceLastLogin: time.Minute * 30})
}
