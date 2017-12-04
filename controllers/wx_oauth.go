package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/chanxuehong/rand"
	"github.com/chanxuehong/session"
	mpoauth2 "github.com/chanxuehong/wechat.v2/mp/oauth2"
	"github.com/chanxuehong/wechat.v2/oauth2"
	"github.com/reechou/holmes"
	"github.com/reechou/real-erp/config"
)

var (
	WeixinOAuth *WxOAuth
)

// check user
type UserInfo struct {
	IfCookie  bool
	AppId     string
	OpenId    string
	Name      string
	AvatarUrl string
	App       int
	Src       int
}

type WxOAuth struct {
	cfg *config.Config
	
	sessionStorage *session.Storage
	oauth2Num      int
	oauth2Endpoint []oauth2.Endpoint
	oauth2Client   []*oauth2.Client
}

func (self *WxOAuth) initWxOauth() {
	self.sessionStorage = session.New(20*60, 60*60)
	self.oauth2Num = len(self.cfg.WxOauth.WxAppId)
	self.oauth2Endpoint = make([]oauth2.Endpoint, self.oauth2Num)
	self.oauth2Client = make([]*oauth2.Client, self.oauth2Num)
	for i := 0; i < self.oauth2Num; i++ {
		self.oauth2Endpoint[i] = mpoauth2.NewEndpoint(
			self.cfg.WxOauth.WxAppId[i],
			self.cfg.WxOauth.WxAppSecret[i])
		self.oauth2Client[i] = &oauth2.Client{
			Endpoint: self.oauth2Endpoint[i],
		}
	}
}

func (self *WxOAuth) checkUserBase(w http.ResponseWriter, r *http.Request) (ui *UserInfo, ifRedirect bool, ifCheckOK bool) {
	ui = &UserInfo{}
	ifCheckOK = true
	var cookieKey string

	defer func() {
		if ui.OpenId != "" {
			// set cookie
			http.SetCookie(w, &http.Cookie{
				Name:    cookieKey,
				Value:   ui.OpenId,
				Path:    "/",
				Expires: time.Now().Add(time.Hour),
			})
		}
	}()

	queryValues, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		holmes.Error("url parse query error: %v", err)
		ifCheckOK = false
		return
	}

	appIdx := 0
	app := queryValues.Get("app")
	if app != "" {
		appIdx, err = strconv.Atoi(app)
		if err != nil {
			holmes.Error("strconv app[%s] error: %v", app, err)
			ifCheckOK = false
			return
		}
	}
	if appIdx >= self.oauth2Num {
		holmes.Error("app idx[%d] confignum[%d] is not ok", appIdx, self.oauth2Num)
		ifCheckOK = false
		return
	}
	
	src := 0
	srcVal := queryValues.Get("src")
	if srcVal != "" {
		src, err = strconv.Atoi(srcVal)
		if err != nil {
			holmes.Error("strconv src[%s] error: %v", app, err)
			ifCheckOK = false
			return
		}
	}
	ui.Src = src

	ui.App = appIdx
	ui.AppId = self.cfg.WxOauth.WxAppId[appIdx]
	cookieKey = fmt.Sprintf("user-%d", appIdx)
	// get cookie
	cookie, err := r.Cookie(cookieKey)
	if err == nil {
		ui.IfCookie = true
		ui.OpenId = cookie.Value
		holmes.Debug("get user[%s] from cookie", ui.OpenId)
		return
	}

	code := queryValues.Get("code")
	if code == "" {
		state := string(rand.NewHex())
		redirectUrl := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())
		AuthCodeURL := mpoauth2.AuthCodeURL(self.cfg.WxOauth.WxAppId[appIdx],
			redirectUrl,
			self.cfg.WxOauth.Oauth2ScopeBase, state)
		ifRedirect = true
		http.Redirect(w, r, AuthCodeURL, http.StatusFound)
		return
	}

	token, err := self.oauth2Client[appIdx].ExchangeToken(code)
	if err != nil {
		ifRedirect = true
		http.Redirect(w, r, fmt.Sprintf("http://%s%s", r.Host, r.URL.Path), http.StatusFound)
		return
	}
	ui.OpenId = token.OpenId
	return
}

func (self *WxOAuth) checkUser(w http.ResponseWriter, r *http.Request) (ui *UserInfo, ifRedirect bool, ifCheckOK bool) {
	ui = &UserInfo{}
	ifCheckOK = true
	var cookieKey string
	
	defer func() {
		if ui.OpenId != "" {
			// set cookie
			http.SetCookie(w, &http.Cookie{
				Name:    cookieKey,
				Value:   ui.OpenId,
				Path:    "/",
				Expires: time.Now().Add(time.Hour),
			})
		}
	}()
	
	queryValues, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		holmes.Error("url parse query error: %v", err)
		ifCheckOK = false
		return
	}
	
	appIdx := 0
	app := queryValues.Get("app")
	if app != "" {
		appIdx, err = strconv.Atoi(app)
		if err != nil {
			holmes.Error("strconv app[%s] error: %v", app, err)
			ifCheckOK = false
			return
		}
	}
	if appIdx >= self.oauth2Num {
		holmes.Error("app idx[%d] confignum[%d] is not ok", appIdx, self.oauth2Num)
		ifCheckOK = false
		return
	}
	
	src := 0
	srcVal := queryValues.Get("src")
	if srcVal != "" {
		src, err = strconv.Atoi(srcVal)
		if err != nil {
			holmes.Error("strconv src[%s] error: %v", app, err)
			ifCheckOK = false
			return
		}
	}
	ui.Src = src
	
	ui.App = appIdx
	ui.AppId = self.cfg.WxOauth.WxAppId[appIdx]
	cookieKey = fmt.Sprintf("user-%d", appIdx)
	
	// get cookie
	cookie, err := r.Cookie(cookieKey)
	if err == nil {
		ui.IfCookie = true
		ui.OpenId = cookie.Value
		holmes.Debug("get user[%s] from cookie", ui.OpenId)
		return
	}
	
	code := queryValues.Get("code")
	if code == "" {
		state := string(rand.NewHex())
		redirectUrl := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())
		AuthCodeURL := mpoauth2.AuthCodeURL(self.cfg.WxOauth.WxAppId[appIdx],
			redirectUrl,
			self.cfg.WxOauth.Oauth2ScopeUser, state)
		ifRedirect = true
		http.Redirect(w, r, AuthCodeURL, http.StatusFound)
		return
	}
	
	token, err := self.oauth2Client[appIdx].ExchangeToken(code)
	if err != nil {
		ifRedirect = true
		http.Redirect(w, r, fmt.Sprintf("http://%s%s", r.Host, r.URL.Path), http.StatusFound)
		return
	}
	
	userinfo, err := mpoauth2.GetUserInfo(token.AccessToken, token.OpenId, "", nil)
	if err != nil {
		holmes.Error("get user info error: %v", err)
		return
	}
	holmes.Debug("user info: %+v", userinfo)
	ui.OpenId = userinfo.OpenId
	ui.Name = userinfo.Nickname
	ui.AvatarUrl = userinfo.HeadImageURL
	return
}

func InitWxOAuth(cfg *config.Config) {
	WeixinOAuth = &WxOAuth{cfg: cfg}
	WeixinOAuth.initWxOauth()
}
