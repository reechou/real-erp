package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/qor/media/media_library"
	"github.com/qor/validations"
	"github.com/reechou/holmes"
)

type Product struct {
	gorm.Model

	Name        string
	Code        string
	CategoryId  uint
	Category    Category
	MainImage   media_library.MediaBox
	Price       float32
	Description string `sql:"size:2000"`

	Variations []ProductVariation
}

func (product Product) DefaultPath() string {
	if len(product.Code) > 0 {
		return fmt.Sprintf("/products/%s", product.Code)
	}
	return "/"
}

func (product Product) MainImageURL(styles ...string) string {
	style := "main"
	if len(styles) > 0 {
		style = styles[0]
	}

	if len(product.MainImage.Files) > 0 {
		return product.MainImage.URL(style)
	}

	return "/images/default_product.png"
}

func (product Product) Validate(db *gorm.DB) {
	if strings.TrimSpace(product.Name) == "" {
		db.AddError(validations.NewError(product, "Name", "Name can not be empty"))
	}

	if strings.TrimSpace(product.Code) == "" {
		db.AddError(validations.NewError(product, "Code", "Code can not be empty"))
	}
}

type ProductImage struct {
	gorm.Model
	Title        string
	Category     Category
	CategoryID   uint
	SelectedType string
	File         media_library.MediaLibraryStorage `sql:"size:4294967295;" media_library:"url:/system/{{class}}/{{primary_key}}/{{column}}.{{extension}}"`
}

func (productImage ProductImage) Validate(db *gorm.DB) {
	if strings.TrimSpace(productImage.Title) == "" {
		db.AddError(validations.NewError(productImage, "Title", "Title can not be empty"))
	}
}

func (productImage *ProductImage) SetSelectedType(typ string) {
	productImage.SelectedType = typ
}

func (productImage *ProductImage) GetSelectedType() string {
	return productImage.SelectedType
}

func (productImage *ProductImage) ScanMediaOptions(mediaOption media_library.MediaOption) error {
	if bytes, err := json.Marshal(mediaOption); err == nil {
		return productImage.File.Scan(bytes)
	} else {
		return err
	}
}

func (productImage *ProductImage) GetMediaOption() (mediaOption media_library.MediaOption) {
	mediaOption.Video = productImage.File.Video
	mediaOption.FileName = productImage.File.FileName
	mediaOption.URL = productImage.File.URL()
	mediaOption.OriginalURL = productImage.File.URL("original")
	mediaOption.CropOptions = productImage.File.CropOptions
	mediaOption.Sizes = productImage.File.GetSizes()
	mediaOption.Description = productImage.File.Description
	return
}

type ProductVariation struct {
	gorm.Model
	ProductID uint
	Product   Product

	SKU               string
	Price             float32
	AvailableQuantity uint
}

func ProductVariations() []ProductVariation {
	variations := []ProductVariation{}
	if err := DB.Preload("Product.Category").Preload("Product").Find(&variations).Error; err != nil {
		holmes.Fatal("query productVariations (%v) failure, got err %v", variations, err)
		return variations
	}
	return variations
}

func (productVariation ProductVariation) ProductVariationInfo() string {
	return fmt.Sprintf("%s (￥%.2f-%d)", productVariation.SKU, productVariation.Price, productVariation.AvailableQuantity)
}

func (productVariation ProductVariation) Stringify() string {
	//holmes.Debug("Product variation Stringify: %v", productVariation)
	if product := productVariation.Product; product.ID != 0 {
		return fmt.Sprintf("【%s】%s (%s-%s)", product.Category.Name, product.Name, product.Code, productVariation.SKU)
	}
	return fmt.Sprint(productVariation.ID)
}

type ProductPurchase struct {
	gorm.Model
	ProductVariationID uint
	ProductVariation   ProductVariation

	Quantity uint
}

func (purchase *ProductPurchase) AfterCreate(tx *gorm.DB) (err error) {
	err = tx.First(&purchase.ProductVariation, purchase.ProductVariationID).Error
	if err != nil {
		return
	}
	purchase.ProductVariation.AvailableQuantity += purchase.Quantity
	err = tx.Select("available_quantity").Save(&purchase.ProductVariation).Error
	return
}
