package routes

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi"
	"github.com/qor/publish2"
	"github.com/qor/qor"
	"github.com/qor/qor/utils"
	"github.com/qor/wildcard_router"

	"github.com/reechou/real-erp/admin/bindatafs"
	"github.com/reechou/real-erp/auth"
	"github.com/reechou/real-erp/config"
	"github.com/reechou/real-erp/models"
	"github.com/reechou/real-erp/controllers"
)

var rootMux *http.ServeMux
var WildcardRouter *wildcard_router.WildcardRouter

func Router() *http.ServeMux {
	if rootMux == nil {
		router := chi.NewRouter()

		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var (
					tx         = models.DB
					qorContext = &qor.Context{Request: req, Writer: w}
				)

				if locale := utils.GetLocale(qorContext); locale != "" {
					tx = tx.Set("l10n:locale", locale)
				}

				ctx := context.WithValue(req.Context(), utils.ContextDBName, publish2.PreviewByDB(tx, qorContext))
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})

		router.Get("/", HomeIndex)
		
		router.Get("/agency/{mp}", controller.AgencyMp)
		router.Get("/agency/l/index", controller.AgencyIndex)

		rootMux = http.NewServeMux()

		rootMux.Handle("/auth/", auth.Auth.NewServeMux())
		rootMux.Handle("/system/", utils.FileServer(http.Dir(filepath.Join(config.Root, "public"))))
		assetFS := bindatafs.AssetFS.FileServer(http.Dir("public"), "javascripts", "stylesheets", "images", "dist", "fonts", "vendors", "mp")
		for _, path := range []string{"javascripts", "stylesheets", "images", "dist", "fonts", "vendors", "mp"} {
			rootMux.Handle(fmt.Sprintf("/%s/", path), assetFS)
		}

		WildcardRouter = wildcard_router.New()
		WildcardRouter.MountTo("/", rootMux)
		WildcardRouter.AddHandler(router)
	}
	return rootMux
}

func HomeIndex(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, "/admin", http.StatusSeeOther)
}
