package controller

import (
	"net/http"
	"io"
	"encoding/json"
	"path/filepath"
	
	"github.com/reechou/real-erp/models"
	"github.com/reechou/real-erp/utils"
	"github.com/reechou/real-erp/config"
	"github.com/reechou/holmes"
	"github.com/jinzhu/gorm"
)

func AgencyMp(w http.ResponseWriter, req *http.Request) {
	var (
		MP = utils.URLParam("mp", req)
	)
	http.ServeFile(w, req, filepath.Join(config.Root, "public", "mp", MP))
	//http.Redirect(w, req, "/mp/"+MP, http.StatusFound)
	return
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
	)
	
	account, err := checkAccount(userinfo, tx)
	if err != nil {
		holmes.Error("check account error: %v", err)
		io.WriteString(w, "system error.")
		return
	}
	//if account.AgencyId == 0 {
	//	// redirect to agency sign
	//	return
	//}
	account.AgencyId = 5
	
	tx.Where("agency_id = ?", account.AgencyId).Preload("AgencyProductQuantities").Find(&agencyLevels)
	for i := 0; i < len(agencyLevels); i++ {
		tx.First(&agencyLevels[i].Category, agencyLevels[i].CategoryID)
		tx.First(&agencyLevels[i].AgencyLevelConfig, agencyLevels[i].AgencyLevelConfigID)
		for j := 0; j < len(agencyLevels[i].AgencyProductQuantities); j++ {
			tx.First(&agencyLevels[i].AgencyProductQuantities[j].ProductVariation, agencyLevels[i].AgencyProductQuantities[j].ProductVariationID)
		}
	}
	
	b, _ := json.Marshal(agencyLevels)
	w.Write(b)
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
	}
	return account, nil
}
