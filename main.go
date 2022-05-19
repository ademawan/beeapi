package main

import (
	_ "beeapi/routers"

	"github.com/astaxie/beego"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	orm.RegisterDriver("mysql", orm.DRMySQL)

	orm.RegisterDataBase(`default`, "mysql", "root:adol1122@/beeweb?charset=utf8")
}

func main() {
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}
	orm.Debug = true
	//=========////migrate table
	// name := "default"

	// // Drop table and re-create.
	// force := true

	// // Print log.
	// verbose := true

	// // Error.
	// err := orm.RunSyncdb(name, force, verbose)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	//migrate table
	beego.Run()
}
