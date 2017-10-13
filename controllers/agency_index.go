package controller

import (
	"net/http"
	"io"
	"encoding/json"
	
	"github.com/reechou/real-erp/models"
	"github.com/reechou/real-erp/utils"
	"github.com/reechou/holmes"
	"github.com/jinzhu/gorm"
)

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
	if account.AgencyId == 0 {
		// redirect to agency sign
		return
	}
	
	tx.Joins("JOIN categories ON categories.id = agency_levels.category_id").Where("agency_id = ?", account.AgencyId).Preload("AgencyProductQuantities").Find(&agencyLevels)
	
	b, _ := json.Marshal(agencyLevels)
	w.Write(b)
}

func checkAccount(userinfo *UserInfo, tx *gorm.DB) (*models.AgencyAccount, error) {
	account := new(models.AgencyAccount)
	var err error
	if err = tx.Where("app_id = ? AND open_id = ?", userinfo.AppId, userinfo.OpenId).First(account).Error; err != nil {
		holmes.Error("check account get error: %v", err)
		return nil, err
	}
	if account.ID == 0 {
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
