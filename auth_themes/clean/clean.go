package clean

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"strings"

	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/auth/providers/password"
	"github.com/qor/i18n"
	"github.com/qor/qor"
	"github.com/qor/qor/utils"
	"github.com/qor/render"
	"github.com/qor/session"
)

// ErrPasswordConfirmationNotMatch password confirmation not match error
var ErrPasswordConfirmationNotMatch = errors.New("password confirmation doesn't match password")

// New initialize clean theme
func New(config *auth.Config) *auth.Auth {
	if config == nil {
		config = &auth.Config{}
	}
	config.ViewPaths = append(config.ViewPaths, "github.com/reechou/real-erp/auth_themes/clean/views")

	if config.DB == nil {
		fmt.Print("Please configure *gorm.DB for Auth theme clean")
	}

	if config.Render == nil {
		config.Render = render.New(&render.Config{
			FuncMapMaker: func(render *render.Render, req *http.Request, w http.ResponseWriter) template.FuncMap {
				return template.FuncMap{
					"t": func(key string, args ...interface{}) template.HTML {
						return i18n.New().T(utils.GetLocale(&qor.Context{Request: req}), key, args...)
					},
				}
			},
		})
	}

	Auth := auth.New(config)

	Auth.RegisterProvider(password.New(&password.Config{
		Confirmable: false,
		RegisterHandler: func(context *auth.Context) (*claims.Claims, error) {
			context.Request.ParseForm()

			if context.Request.Form.Get("confirm_password") != context.Request.Form.Get("password") {
				return nil, ErrPasswordConfirmationNotMatch
			}

			return RealErpRegisterHandler(context)
		},
	}))

	if Auth.Config.DB != nil {
		// Migrate Auth Identity model
		Auth.Config.DB.AutoMigrate(Auth.Config.AuthIdentityModel)
	}
	return Auth
}

var RealErpRegisterHandler = func(context *auth.Context) (*claims.Claims, error) {
	var (
		err         error
		currentUser interface{}
		schema      auth.Schema
		authInfo    auth_identity.Basic
		req         = context.Request
		tx          = context.Auth.GetDB(req)
		provider, _ = context.Provider.(*password.Provider)
	)

	req.ParseForm()
	if req.Form.Get("login") == "" {
		return nil, auth.ErrInvalidAccount
	}

	if req.Form.Get("password") == "" {
		return nil, auth.ErrInvalidPassword
	}

	authInfo.Provider = provider.GetName()
	authInfo.UID = strings.TrimSpace(req.Form.Get("login"))

	if !tx.Model(context.Auth.AuthIdentityModel).Where(authInfo).Scan(&authInfo).RecordNotFound() {
		return nil, auth.ErrInvalidAccount
	}

	if authInfo.EncryptedPassword, err = provider.Encryptor.Digest(strings.TrimSpace(req.Form.Get("password"))); err == nil {
		schema.Provider = authInfo.Provider
		schema.UID = authInfo.UID
		schema.Email = authInfo.UID
		schema.Name = authInfo.UID
		schema.RawInfo = req

		currentUser, authInfo.UserID, err = context.Auth.UserStorer.Save(&schema, context)
		if err != nil {
			return nil, err
		}

		// create auth identity
		authIdentity := reflect.New(utils.ModelType(context.Auth.Config.AuthIdentityModel)).Interface()
		if err = tx.Where(authInfo).FirstOrCreate(authIdentity).Error; err == nil {
			if provider.Config.Confirmable {
				context.SessionStorer.Flash(context.Writer, req, session.Message{Message: password.ConfirmFlashMessage, Type: "success"})
				err = provider.Config.ConfirmMailer(schema.Email, context, authInfo.ToClaims(), currentUser)
			}

			return authInfo.ToClaims(), err
		}
	}

	return nil, err
}
