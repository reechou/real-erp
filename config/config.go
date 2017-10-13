package config

import (
	"flag"
	"fmt"
	"html/template"
	"os"

	"github.com/go-ini/ini"
	"github.com/microcosm-cc/bluemonday"
	"github.com/qor/redirect_back"
	"github.com/qor/render"
	"github.com/qor/session/manager"
)

var (
	Root         = os.Getenv("GOPATH") + "/src/github.com/reechou/real-erp"
	View         *render.Render
	RedirectBack = redirect_back.New(&redirect_back.Config{
		SessionManager:  manager.SessionManager,
		IgnoredPrefixes: []string{"/auth"},
	})
)

type DBInfo struct {
	Adapter string
	User    string
	Pass    string
	Host    string
	DBName  string
}

type WxOauth struct {
	WxAppId         []string
	WxAppSecret     []string
	MchId           []string
	MchApiKey       []string
	Oauth2ScopeBase string
	Oauth2ScopeUser string
	MpVerifyDir     string
}

type Config struct {
	Debug     bool
	Path      string
	Host      string
	Version   string
	IfShowSql bool

	DBInfo
	WxOauth
}

func NewConfig() *Config {
	c := new(Config)
	//initFlag(c)

	//if c.Path == "" {
	//	fmt.Println("server must run with config file, please check.")
	//	os.Exit(0)
	//}

	c.Path = "real-erp.ini"

	cfg, err := ini.Load(c.Path)
	if err != nil {
		fmt.Printf("ini[%s] load error: %v\n", c.Path, err)
		os.Exit(1)
	}
	cfg.BlockMode = false
	err = cfg.MapTo(c)
	if err != nil {
		fmt.Printf("config MapTo error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(c)

	View = render.New(nil)
	htmlSanitizer := bluemonday.UGCPolicy()
	View.RegisterFuncMap("raw", func(str string) template.HTML {
		return template.HTML(htmlSanitizer.Sanitize(str))
	})

	return c
}

func initFlag(c *Config) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	v := fs.Bool("v", false, "Print version and exit")
	fs.StringVar(&c.Path, "c", "", "server config file.")

	fs.Parse(os.Args[1:])
	fs.Usage = func() {
		fmt.Println("Usage: " + os.Args[0] + " -c api.ini")
		fmt.Printf("\nglobal flags:\n")
		fs.PrintDefaults()
	}

	if *v {
		fmt.Println("version: 0.0.1")
		os.Exit(0)
	}
}
