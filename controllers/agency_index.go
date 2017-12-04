package controller

import (
	"io"
	"net/http"
	//"encoding/json"
	"path/filepath"

	"github.com/jinzhu/gorm"
	"github.com/reechou/holmes"
	"github.com/reechou/real-erp/config"
	"github.com/reechou/real-erp/models"
	"github.com/reechou/real-erp/utils"
)

func AgencyMp(w http.ResponseWriter, req *http.Request) {
	var (
		MP = utils.URLParam("mp", req)
	)
	http.ServeFile(w, req, filepath.Join(config.Root, "public", "mp", MP))
	return
}

func AgencySign(w http.ResponseWriter, req *http.Request) {
	userinfo, ifRedirect, ifCheckOK := WeixinOAuth.checkUser(w, req)
	if !ifCheckOK {
		io.WriteString(w, "system error.")
		return
	}
	if ifRedirect {
		return
	}
	holmes.Debug("userinfo: %v", userinfo)
	
	type AgencySignInfo struct {
		AppId  string
		OpenId string
		SrcId  int
	}

	var (
		account *models.AgencyAccount
		tx      = utils.GetDB(req)
		err     error
	)

	account, err = checkAccount(userinfo, tx)
	if err != nil {
		holmes.Error("check account error: %v", err)
		io.WriteString(w, "system error.")
		return
	}
	holmes.Debug("sign account: %v", account)
	
	agency := new(models.Agency)
	rowsAffected := tx.Where("agency_account_id = ?", account.ID).First(agency).RowsAffected
	if rowsAffected != 0 {
		// redirect to agency index
		http.Redirect(w, req, "/agency/l/index", http.StatusFound)
		return
	}

	config.View.Execute("/agency/register", map[string]interface{}{
		"Account":  account,
		"SignInfo": &AgencySignInfo{AppId: userinfo.AppId, OpenId: userinfo.OpenId, SrcId: userinfo.Src},
	}, req, w)
}

func AgencyIndex(w http.ResponseWriter, req *http.Request) {
	userinfo, ifRedirect, ifCheckOK := WeixinOAuth.checkUser(w, req)
	if !ifCheckOK {
		io.WriteString(w, "system error.")
		return
	}
	if ifRedirect {
		return
	}
	holmes.Debug("userinfo: %v", userinfo)

	var (
		agencyLevels []models.AgencyLevel
		tx           = utils.GetDB(req)
		subAgencies  []models.Agency
	)

	account, err := checkAccount(userinfo, tx)
	if err != nil {
		holmes.Error("check account error: %v", err)
		io.WriteString(w, "system error.")
		return
	}

	agency := new(models.Agency)
	rowsAffected := tx.Where("agency_account_id = ?", account.ID).First(agency).RowsAffected
	if rowsAffected == 0 {
		// redirect to agency sign
		http.Redirect(w, req, "/agency/l/sign", http.StatusFound)
		return
	}
	agency.AgencyAccount = *account
	if agency.SuperiorID != 0 {
		agency.Superior = new(models.Agency)
		tx.Where("id = ?", agency.SuperiorID).Preload("AgencyAccount").First(agency.Superior)
		holmes.Debug("superior: %v", agency.Superior)
	}
	tx.Where("superior_id = ?", agency.ID).Preload("AgencyAccount").Find(&subAgencies)
	//holmes.Debug("subAgencies: %v", subAgencies)

	tx.Where("agency_id = ?", agency.ID).Preload("AgencyProductQuantities").Find(&agencyLevels)
	for i := 0; i < len(agencyLevels); i++ {
		tx.First(&agencyLevels[i].Category, agencyLevels[i].CategoryID)
		agencyLevels[i].Category.ImageUrl = agencyLevels[i].Category.MainImageURL()
		tx.First(&agencyLevels[i].AgencyLevelConfig, agencyLevels[i].AgencyLevelConfigID)
		for j := 0; j < len(agencyLevels[i].AgencyProductQuantities); j++ {
			tx.First(&agencyLevels[i].AgencyProductQuantities[j].ProductVariation, agencyLevels[i].AgencyProductQuantities[j].ProductVariationID)
			tx.First(&agencyLevels[i].AgencyProductQuantities[j].ProductVariation.Product, agencyLevels[i].AgencyProductQuantities[j].ProductVariation.ProductID)
		}
	}
	//holmes.Debug("111111 agency level: %+v", agencyLevels)

	//if account.AgencyId == 0 {
	//	// redirect to agency sign
	//	return
	//}
	//account.AgencyId = 5
	//
	//tx.First(&account.Agency, account.AgencyId)
	//if account.Agency.SuperiorID != 0 {
	//	account.Agency.Superior = new(models.Agency)
	//	tx.First(account.Agency.Superior, account.Agency.SuperiorID)
	//}
	//tx.Where("superior_id = ?", account.AgencyId).Find(&subAgencies)
	//
	//tx.Where("agency_id = ?", account.AgencyId).Preload("AgencyProductQuantities").Find(&agencyLevels)
	//for i := 0; i < len(agencyLevels); i++ {
	//	tx.First(&agencyLevels[i].Category, agencyLevels[i].CategoryID)
	//	tx.First(&agencyLevels[i].AgencyLevelConfig, agencyLevels[i].AgencyLevelConfigID)
	//	for j := 0; j < len(agencyLevels[i].AgencyProductQuantities); j++ {
	//		tx.First(&agencyLevels[i].AgencyProductQuantities[j].ProductVariation, agencyLevels[i].AgencyProductQuantities[j].ProductVariationID)
	//	}
	//}

	//b, _ := json.Marshal(agencyLevels)
	//w.Write(b)

	config.View.Execute("/agency/index", map[string]interface{}{
		"Account":      agency,
		"SubAgencies":  subAgencies,
		"AgencyLevels": agencyLevels,
	}, req, w)
}

func AgencyOrderShipping(w http.ResponseWriter, req *http.Request) {
	userinfo, ifRedirect, ifCheckOK := WeixinOAuth.checkUser(w, req)
	if !ifCheckOK {
		io.WriteString(w, "system error.")
		return
	}
	if ifRedirect {
		return
	}
	holmes.Debug("userinfo: %v", userinfo)
	
	type AgencyOrderShippingOwnInfo struct {
		AgencyId   string
		ProductId  string
		QuantityId string
	}

	var (
		agencyId   = utils.URLParam("agency", req)
		productId  = utils.URLParam("product", req)
		quantityId = utils.URLParam("quantity", req)
		tx         = utils.GetDB(req)
		subAgencies []models.Agency
	)
	holmes.Debug("product: %v %v %v", agencyId, productId, quantityId)

	productVariation := new(models.ProductVariation)
	tx.First(productVariation, productId)
	agencyProductQuantity := new(models.AgencyProductQuantity)
	tx.First(agencyProductQuantity, quantityId)
	tx.Where("superior_id = ?", agencyId).Find(&subAgencies)
	holmes.Debug("product detail: %v %v %v", productVariation, agencyProductQuantity, subAgencies)

	config.View.Execute("/agency/order_shipping", map[string]interface{}{
		"OwnInfo":               &AgencyOrderShippingOwnInfo{AgencyId: agencyId, ProductId: productId, QuantityId: quantityId},
		"ProductVariation":      productVariation,
		"AgencyProductQuantity": agencyProductQuantity,
		"SubAgencies":           subAgencies,
	}, req, w)
}

func checkAccount(userinfo *UserInfo, tx *gorm.DB) (*models.AgencyAccount, error) {
	account := new(models.AgencyAccount)
	var err error
	rowsAffected := tx.Where("app_id = ? AND open_id = ?", userinfo.AppId, userinfo.OpenId).First(account).RowsAffected
	if rowsAffected == 0 {
		account.AppId = userinfo.AppId
		account.OpenId = userinfo.OpenId
		account.Name = userinfo.Name
		account.AvatarUrl = userinfo.AvatarUrl
		if err = tx.Save(account).Error; err != nil {
			holmes.Error("save account error: %v", err)
			return nil, err
		}
	} else {
		if userinfo.Name == "" && userinfo.AvatarUrl == "" {
			return account, nil
		}
		if account.Name != userinfo.Name || account.AvatarUrl != userinfo.AvatarUrl {
			account.Name = userinfo.Name
			account.AvatarUrl = userinfo.AvatarUrl
			if err = tx.Save(account).Error; err != nil {
				holmes.Error("save account error: %v", err)
				return nil, err
			}
		}
	}
	return account, nil
}
