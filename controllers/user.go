package controllers

import (
	"beeapi/models"
	"beeapi/utils/datatables"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/astaxie/beego"
)

type RequestBody struct {
	Ctx          url.Values `json:"ctx"`
	TableName    string     `json:"tablename"`
	Columns      []string   `json:"column"`
	Order        []string   `json:"order"`
	SearchFilter []string   `json:"searchfilter"`
}

// Operations about Users
type UserController struct {
	beego.Controller
}

func (u *UserController) GetAjax() {
	// seeHeader := u.Ctx.Request.Header

	var reqBody RequestBody
	json.Unmarshal(u.Ctx.Input.RequestBody, &reqBody)

	var err error
	var Qtab datatables.Data
	Qtab.Ctx = reqBody.Ctx
	// Qtab.DBName = "default"
	Qtab.TableName = reqBody.TableName //modles tables name
	Qtab.Columns = reqBody.Columns     //datatables columns arrange
	Qtab.Order = reqBody.Order
	Qtab.SearchFilter = reqBody.SearchFilter                        //datatables filter
	datatables.RegisterColumns[Qtab.TableName] = new([]models.User) //register result
	if err != nil {
		fmt.Println(err)
		return
	}
	rs, _ := Qtab.Table()

	u.Data["json"] = rs

	u.ServeJSON()
}
func (u *UserController) GetAll() {
	// seeHeader := u.Ctx.Request.Header
	// fmt.Println(seeHeader)

	var err error
	var Qtab datatables.Data
	// Qtab.DBName = "default"
	Qtab.TableName = "user"                                  //modles tables name
	Qtab.Columns = []string{"id", "nama", "alamat", "email"} //datatables columns arrange
	Qtab.Order = []string{"nama"}
	Qtab.SearchFilter = []string{"nama", "alamat"}                  //datatables filter
	datatables.RegisterColumns[Qtab.TableName] = new([]models.User) //register result
	if err != nil {
		fmt.Println(err)
		return
	}
	rs, _ := Qtab.Table()

	u.Data["json"] = rs

	u.ServeJSON()
}

func (u *UserController) Get() {
	uid := u.Ctx.Input.Param(":id")
	if uid != "" {
		user, err := models.GetUserById(uid)
		if err != nil {
			data := Response{Code: http.StatusInternalServerError, Message: "gagal", Data: nil}
			u.Ctx.Output.Status = http.StatusInternalServerError
			u.Data["json"] = data
		} else {
			data := Response{Code: http.StatusOK, Message: "berhasil", Data: user}
			u.Ctx.Output.Status = http.StatusOK
			u.Data["json"] = data
		}
	}

	u.ServeJSON()
}

func (u *UserController) Put() {
	uid := u.Ctx.Input.Param(":id")
	if uid != "" {
		var user models.User
		json.Unmarshal(u.Ctx.Input.RequestBody, &user)
		user.Uid = uid
		fmt.Println("UPdate====", user, "==========")
		uu, err := models.UpdateUserById(&user)
		if err != nil {
			data := Response{Code: http.StatusInternalServerError, Message: "gagal", Data: nil}
			u.Ctx.Output.Status = http.StatusInternalServerError
			u.Data["json"] = data
		} else {
			data := Response{Code: http.StatusOK, Message: "berhasil", Data: uu.Uid}
			u.Ctx.Output.Status = http.StatusOK
			u.Data["json"] = data
		}
	}
	u.ServeJSON()
}

func (u *UserController) Delete() {
	uid := u.Ctx.Input.Param(":id")

	err := models.DeleteUser(uid)

	if err != nil {
		data := Response{Code: http.StatusInternalServerError, Message: "gagal", Data: nil}
		u.Ctx.Output.Status = http.StatusInternalServerError
		u.Data["json"] = data
	} else {
		data := Response{Code: http.StatusOK, Message: "berhasil", Data: nil}
		u.Ctx.Output.Status = http.StatusOK
		u.Data["json"] = data
	}

	u.ServeJSON()
}
