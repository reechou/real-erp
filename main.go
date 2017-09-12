package main

import (
	"github.com/reechou/real-erp/config"
	"github.com/reechou/real-erp/controllers"
)

func main() {
	controllers.NewLogic(config.NewConfig()).Run()
}
