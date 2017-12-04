package models

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/qor/validations"
	"github.com/qor/media/media_library"
)

type Category struct {
	gorm.Model
	Name string
	Code string
	
	MainImage media_library.MediaBox
	ImageUrl  string `gorm:"-"`
}

func (category Category) Validate(db *gorm.DB) {
	if strings.TrimSpace(category.Name) == "" {
		db.AddError(validations.NewError(category, "Name", "Name can not be empty"))
	}
}

func (category Category) DefaultPath() string {
	if len(category.Code) > 0 {
		return fmt.Sprintf("/category/%s", category.Code)
	}
	return "/"
}

func (category Category) MainImageURL(styles ...string) string {
	style := "main"
	if len(styles) > 0 {
		style = styles[0]
	}
	
	if len(category.MainImage.Files) > 0 {
		return category.MainImage.URL(style)
	}
	
	return "/images/default_product.png"
}
