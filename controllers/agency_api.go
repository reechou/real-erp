package controller

import (
	"encoding/json"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/now"
	"github.com/reechou/holmes"
	"github.com/reechou/real-erp/models"
	"github.com/reechou/real-erp/utils"
)

type AgencySignReq struct {
	AppId    string `json:"appId"`
	OpenId   string `json:"openId"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	IDCard   string `json:"idcard"`
	Wechat   string `json:"wechat"`
	Superior uint   `json:"superior"`
}

type AgencyOrdersReq struct {
	AgencyID           uint   `json:"agencyId"`
	CategoryID         uint   `json:"categoryId"`
	ProductVariationID uint   `json:"productVariationId"`
	OrderStatus        uint   `json:"orderStatus"`
	OrderDate          string `json:"orderDate"`
}

type AgencyOrderShippingReq struct {
	OwnAgencyID             uint    `json:"ownAgencyID"`
	ProductVariationID      uint    `json:"productVariationID"`
	AgencyProductQuantityID uint    `json:"agencyProductQuantityId"`
	AgencyId                uint    `json:"agencyId"`
	Quantity                uint    `json:"quantity"`
	Price                   float32 `json:"price"`
	Name                    string  `json:"name"`
	Phone                   string  `json:"phone"`
	Express                 string  `json:"express"`
	TrackingNumber          string  `json:"trackingNumber"`
}

type AgencyApiRsp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

const (
	AGENCY_API_RSP_OK = iota
	AGENCY_API_SYSTEM_ERROR
	AGENCY_API_USER_ERROR
)

func ApiAgencySign(w http.ResponseWriter, req *http.Request) {
	var (
		tx  = utils.GetDB(req)
		err error
	)

	response := new(AgencyApiRsp)
	response.Code = AGENCY_API_RSP_OK
	defer func() {
		json.NewEncoder(w).Encode(response)
		return
	}()

	request := new(AgencySignReq)
	if err = json.NewDecoder(req.Body).Decode(request); err != nil {
		holmes.Error("json decode error: %v", err)
		response.Code = AGENCY_API_SYSTEM_ERROR
		return
	}

	account := new(models.AgencyAccount)
	rowsAffected := tx.Where("app_id = ? AND open_id = ?", request.AppId, request.OpenId).First(account).RowsAffected
	if rowsAffected == 0 {
		account.AppId = request.AppId
		account.OpenId = request.OpenId
		if err = tx.Save(account).Error; err != nil {
			holmes.Error("save account error: %v", err)
			response.Code = AGENCY_API_SYSTEM_ERROR
			return
		}
	}
	agency := new(models.Agency)
	rowsAffected = tx.Where("phone = ?", request.Phone).First(agency).RowsAffected
	var ifChangeAgency bool
	if rowsAffected != 0 {
		if agency.AgencyAccountID == 0 {
			agency.AgencyAccountID = account.ID
			ifChangeAgency = true
		}
		if request.Name != "" && request.Name != agency.Name {
			agency.Name = request.Name
			ifChangeAgency = true
		}
		if request.IDCard != "" && request.IDCard != agency.IDCardNumber {
			agency.IDCardNumber = request.IDCard
			ifChangeAgency = true
		}
		if request.Wechat != "" && request.Wechat != agency.Wechat {
			agency.Wechat = request.Wechat
			ifChangeAgency = true
		}
	} else {
		//rowsAffected = tx.Where("agency_account_id = ?", account.ID).First(agency).RowsAffected
		//if rowsAffected != 0 {
		//
		//}
		agency.Name = request.Name
		agency.Phone = request.Phone
		agency.IDCardNumber = request.IDCard
		agency.Wechat = request.Wechat
		agency.SuperiorID = request.Superior
		agency.AgencyAccountID = account.ID
		ifChangeAgency = true
	}
	if ifChangeAgency {
		if err = tx.Save(agency).Error; err != nil {
			holmes.Error("save agency error: %v", err)
			response.Code = AGENCY_API_SYSTEM_ERROR
			return
		}
	}
}

func ApiAgencyOrders(w http.ResponseWriter, req *http.Request) {
	var (
		tx           = utils.GetDB(req)
		err          error
		agencyOrders []models.AgencyOrder
	)

	response := new(AgencyApiRsp)
	response.Code = AGENCY_API_RSP_OK
	defer func() {
		json.NewEncoder(w).Encode(response)
		return
	}()

	request := new(AgencyOrdersReq)
	if err = json.NewDecoder(req.Body).Decode(request); err != nil {
		holmes.Error("json decode error: %v", err)
		response.Code = AGENCY_API_SYSTEM_ERROR
		return
	}

	var txClone *gorm.DB
	if request.OrderDate != "" {
		startDate, err := now.Parse(request.OrderDate)
		if err != nil {
			holmes.Error("parse order date error: %v", err)
			response.Code = AGENCY_API_SYSTEM_ERROR
			return
		}
		endDate := startDate.AddDate(0, 0, 1)
		if request.OrderStatus == 10000 {
			txClone = tx.Where("agency_id = ? AND deleted_at IS NULL AND created_at > ? AND created_at < ?", request.AgencyID, startDate, endDate)
		} else {
			txClone = tx.Where("agency_id = ? AND order_status = ? AND deleted_at IS NULL AND created_at > ? AND created_at < ?",
				request.AgencyID, request.OrderStatus, startDate, endDate)
		}
	} else {
		if request.OrderStatus == 10000 {
			txClone = tx.Where("agency_id = ?", request.AgencyID)
		} else {
			txClone = tx.Where("agency_id = ? AND order_status = ?", request.AgencyID, request.OrderStatus)
		}
	}
	txClone = txClone.Order("id desc")
	if request.CategoryID != 0 {
		txClone = txClone.Preload("AgencyOrderItems.ProductVariation.Product.Category", "id = ?", request.CategoryID)
	} else {
		txClone = txClone.Preload("AgencyOrderItems.ProductVariation.Product.Category")
	}
	txClone = txClone.Preload("AgencyOrderItems.ProductVariation.Product").
		Preload("AgencyOrderItems.ProductVariation")
	if request.ProductVariationID != 0 {
		holmes.Debug("req: %+v", request)
		txClone = txClone.Preload("AgencyOrderItems", "product_variation_id = ?", request.ProductVariationID)
	} else {
		txClone = txClone.Preload("AgencyOrderItems")
	}
	txClone = txClone.Preload("ShippingAddress")
	txClone.Find(&agencyOrders)
	//holmes.Debug("api agency orders: %+v", agencyOrders)
	response.Data = agencyOrders
}

func ApiAgencyOrderShipping(w http.ResponseWriter, req *http.Request) {
	var (
		tx  = utils.GetDB(req)
		err error
	)

	response := new(AgencyApiRsp)
	response.Code = AGENCY_API_RSP_OK
	defer func() {
		json.NewEncoder(w).Encode(response)
		return
	}()

	request := new(AgencyOrderShippingReq)
	if err = json.NewDecoder(req.Body).Decode(request); err != nil {
		holmes.Error("json decode error: %v", err)
		response.Code = AGENCY_API_SYSTEM_ERROR
		return
	}
	holmes.Debug("agency order shipping req: %+v", request)

	tx = tx.Begin()
	address := new(models.Address)
	address.ContactName = request.Name
	address.Phone = request.Phone
	if err = tx.Create(address).Error; err != nil {
		holmes.Error("save address error: %v", err)
		response.Code = AGENCY_API_SYSTEM_ERROR
		return
	}
	// 扣减上级代理库存
	rowsAffected := tx.Table("agency_product_quantities").
		Where("id = ? AND quantity >= ?", request.AgencyProductQuantityID, request.Quantity).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", request.Quantity)).RowsAffected
	if rowsAffected == 0 {
		holmes.Error("AgencyProductQuantity[%d]'s AvailableQuantity < Request's Quantity[%d]", request.AgencyProductQuantityID, request.Quantity)
		response.Code = AGENCY_API_SYSTEM_ERROR
		return
	}
	// 增加上级代理订单
	superAgencyOrderItem := new(models.AgencyOrderItem)
	superAgencyOrderItem.ProductVariationID = request.ProductVariationID
	superAgencyOrderItem.Quantity = request.Quantity
	superAgencyOrderItem.Price = request.Price
	superAgencyOrderItem.OrderStatus = 1
	
	superAgencyOrder := new(models.AgencyOrder)
	superAgencyOrder.AgencyID = request.OwnAgencyID
	superAgencyOrder.Express = request.Express
	superAgencyOrder.TrackingNumber = request.TrackingNumber
	superAgencyOrder.ShippingAddressID = address.ID
	superAgencyOrder.OrderStatus = 1
	superAgencyOrder.AgencyOrderItems = append(superAgencyOrder.AgencyOrderItems, *superAgencyOrderItem)
	if err = tx.Create(superAgencyOrder).Error; err != nil {
		holmes.Error("save super agency order error: %v", err)
		response.Code = AGENCY_API_SYSTEM_ERROR
		tx.Rollback()
		return
	}
	// 增加下级代理订单
	agencyOrderItem := new(models.AgencyOrderItem)
	agencyOrderItem.ProductVariationID = request.ProductVariationID
	agencyOrderItem.Quantity = request.Quantity
	agencyOrderItem.Price = request.Price
	agencyOrderItem.OrderStatus = 0
	
	agencyOrder := new(models.AgencyOrder)
	agencyOrder.AgencyID = request.AgencyId
	agencyOrder.Express = request.Express
	agencyOrder.TrackingNumber = request.TrackingNumber
	agencyOrder.ShippingAddressID = address.ID
	agencyOrder.OrderStatus = 0
	agencyOrder.AgencySrc = request.OwnAgencyID
	agencyOrder.AgencyOrderItems = append(agencyOrder.AgencyOrderItems, *agencyOrderItem)
	if err = tx.Create(agencyOrder).Error; err != nil {
		holmes.Error("save agency order error: %v", err)
		response.Code = AGENCY_API_SYSTEM_ERROR
		tx.Rollback()
		return
	}

	tx.Commit()
}
