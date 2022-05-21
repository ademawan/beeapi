package controllers

import (
	"beeapi/models"
	"beeapi/utils/datatables"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/validation"
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

func (u *UserController) GetAll() {
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
func (u *UserController) GetAjax() {
	defer u.ServeJSON()

	var reqBody RequestBody
	json.Unmarshal(u.Ctx.Input.RequestBody, &reqBody)

	var ctx url.Values = reqBody.Ctx
	start, _ := strconv.Atoi(ctx.Get("start"))
	length, _ := strconv.Atoi(ctx.Get("length"))
	search := ctx.Get("search[value]")
	order_column, _ := strconv.Atoi(ctx.Get("order[0][column]"))
	order_dir := ctx.Get("order[0][dir]")
	draws, _ := strconv.Atoi(ctx.Get("draw"))

	url := make(map[string]interface{})
	url["start"] = start
	url["length"] = length
	url["search"] = search
	url["order_column"] = order_column
	url["order_dir"] = order_dir
	url["draws"] = draws
	url["start"] = start

	Qtab := make(map[string]interface{})
	Qtab["url"] = url
	// Qtab.DBName = "default"
	Qtab["columns"] = reqBody.Columns //datatables columns arrange
	Qtab["order"] = reqBody.Order
	Qtab["searchfilter"] = reqBody.SearchFilter //datatables filter

	userModel := new(models.UserModel)

	users, count, _ := userModel.GetDatatables(Qtab)
	data := map[string]interface{}{
		"draw":            int32(draws),
		"recordsTotal":    count,
		"recordsFiltered": count,
		"data":            users,
	}

	u.Data["json"] = data

}

//====================divide get=======================

func (u *UserController) Get() {
	defer u.ServeJSON()
	output := make(map[string]interface{})
	output["success"] = false

	key := u.Ctx.Input.Param(":id")
	if key == "" {
		u.Ctx.Output.SetStatus(403)
		output["error"] = "User tidak ditemukan"
		output["success"] = true
		u.Data["json"] = output
		return
	}
	userModel := new(models.UserModel)

	user, err := userModel.GetObject(key)
	if err != nil {
		u.Ctx.Output.SetStatus(403)
		output["error"] = "User tidak ditemukan"
		output["success"] = true
		u.Data["json"] = output
		return
	}
	userMap := user.(map[string]interface{})

	delete(userMap, "password")

	output["object"] = userMap
	output["success"] = true
	u.Data["json"] = output

}

func (u *UserController) Put() {
	defer u.ServeJSON()
	output := make(map[string]interface{})
	output["success"] = false
	key := u.Ctx.Input.Param(":id")

	err := u.Ctx.Request.ParseForm()
	if err != nil {
		u.Ctx.Output.SetStatus(404)
		output["error"] = "Data tidak dapat diproses"
		u.Data["json"] = output
		return
	}

	if u.Ctx.Request.PostForm == nil || len(u.Ctx.Request.PostForm) == 0 {
		u.Ctx.Output.SetStatus(403)
		output["error"] = "Data kosong"
		u.Data["json"] = output
		return
	}

	var userData models.User
	if err := u.ParseForm(&userData); err != nil {
		u.Ctx.Output.SetStatus(404)
		output["error"] = "Data tidak dapat diproses"
		u.Data["json"] = output
		return
	}

	nama := strings.TrimSpace(u.GetString("nama"))
	alamat := strings.TrimSpace(u.GetString("alamat"))

	email := strings.TrimSpace(u.GetString("email"))
	password := u.GetString("password")

	valid := validation.Validation{}
	valid.Required(nama, "nama")
	valid.Required(alamat, "alamat")
	if email != "" {
		valid.Email(email, "email")
	}
	valid.MinSize(password, 6, "password")
	if valid.HasErrors() {
		validators := make(map[string]string)
		for _, err := range valid.Errors {
			validators[err.Key] = err.Message
		}
		u.Ctx.Output.SetStatus(403)
		output["validators"] = validators
		output["error"] = "Validasi gagal"
		u.Data["json"] = output
		return
	}
	dataMap := make(map[string]interface{})
	if userData.Nama != "" {
		dataMap["nama"] = userData.Nama
	}
	if userData.Alamat != "" {
		dataMap["alamat"] = userData.Alamat
	}
	userModel := new(models.UserModel)

	user, err := userModel.Update(key, &dataMap)
	userMap := user.(map[string]interface{})
	delete(userMap, "password")

	output["object"] = userMap
	output["success"] = true
	u.Data["json"] = output
}

func (u *UserController) Delete() {
	defer u.ServeJSON()
	output := make(map[string]interface{})
	output["success"] = false
	uid := u.Ctx.Input.Param(":id")
	userModel := new(models.UserModel)

	user, err := userModel.Delete(uid)
	if err != nil {
		u.Ctx.Output.SetStatus(403)
		output["error"] = "User tidak ditemukan"
		output["success"] = true
		u.Data["json"] = output
		return
	}
	userMap := user.(map[string]interface{})
	delete(userMap, "password")

	output["object"] = userMap
	output["success"] = true
	u.Data["json"] = output

}
