package main

import (
	_ "beeapi/routers"

	"github.com/astaxie/beego"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	// orm.RegisterDriver("mysql", orm.DRMySQL)

	// orm.RegisterDataBase(`default`, "mysql", "root:adol1122@/beeweb?charset=utf8")
}

func main() {
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}

	beego.Run()
}
