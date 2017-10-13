package main

import (
	"github.com/reechou/real-erp/config"
	"github.com/reechou/real-erp/main_init"
)

func main() {
	main_init.NewLogic(config.NewConfig()).Run()
}
