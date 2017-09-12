package i18n

import (
	"path/filepath"

	"github.com/qor/i18n"
	"github.com/qor/i18n/backends/database"
	"github.com/qor/i18n/backends/yaml"

	"github.com/reechou/real-erp/config"
	"github.com/reechou/real-erp/models"
)

var I18n *i18n.I18n

func InitI18n() {
	I18n = i18n.New(database.New(models.DB), yaml.New(filepath.Join(config.Root, "locales")))
}
