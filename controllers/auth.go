package controllers

import (
	"beeapi/controllers/middlewares"
	"beeapi/models"
	"encoding/json"
	"errors"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/validation"
)

// type RequestBody struct {
// 	Ctx          url.Values `json:"ctx"`
// 	TableName    string     `json:"tablename"`
// 	Columns      []string   `json:"column"`
// 	Order        []string   `json:"order"`
// 	SearchFilter []string   `json:"searchfilter"`
// }

// Operations about Users
type AuthController struct {
	beego.Controller
}

func (u *AuthController) Login() {
	defer u.ServeJSON()
	output := make(map[string]interface{})
	output["success"] = false
	var userReq LoginRequestFormat
	json.Unmarshal(u.Ctx.Input.RequestBody, &userReq)
	if err := json.Unmarshal(u.Ctx.Input.RequestBody, &userReq); err != nil {
		u.Ctx.Output.SetStatus(400)
		output["error"] = "Data tidak dapat diproses"
		u.Data["json"] = output
		return
	}
	email := strings.TrimSpace(userReq.Email)
	password := userReq.Password

	valid := validation.Validation{}
	valid.Required(email, "email")
	valid.Required(password, "password")

	if email != "" {
		valid.Email(email, "email")
	}
	if valid.HasErrors() {
		validators := make(map[string]string)
		for _, err := range valid.Errors {
			validators[err.Key] = err.Message
		}
		u.Ctx.Output.SetStatus(400)
		output["validators"] = validators
		output["error"] = "Validasi gagal"
		u.Data["json"] = output
		return
	}
	userModel := new(models.UserModel)

	user, err := userModel.Login(userReq.Email)
	if err != nil {
		u.Ctx.Output.SetStatus(500)
		output["error"] = "pengambilan data user gagal"
		u.Data["json"] = output
	}
	userData := user.(map[string]interface{})

	if checked := middlewares.CheckPasswordHash(userReq.Password, userData["password"].(string)); !checked {
		u.Ctx.Output.SetStatus(403)
		output["error"] = "password salah"
		u.Data["json"] = output
	}

	token, err := middlewares.GenerateToken(userData)
	if err != nil {
		u.Ctx.Output.SetStatus(500)
		output["error"] = "gagal generate token"
		u.Data["json"] = output
	}

	userMap := userData

	delete(userMap, "password")

	output["object"] = map[string]interface{}{
		"user":  userMap,
		"token": token,
	}
	output["success"] = true
	u.Data["json"] = output

}

func (u *AuthController) Register() {
	defer u.ServeJSON()
	output := make(map[string]interface{})
	output["success"] = false

	// err := u.Ctx.Request.ParseForm()
	// if err != nil {
	// 	u.Ctx.Output.SetStatus(404)
	// 	output["error"] = "Data tidak dapat diproses"
	// 	u.Data["json"] = output
	// 	return
	// }

	// log.Info(u.Ctx.Request.ParseForm())
	// if u.Ctx.Request.PostForm == nil || len(u.Ctx.Request.PostForm) == 0 {
	// 	u.Ctx.Output.SetStatus(403)
	// 	output["error"] = "Data kosong"
	// 	u.Data["json"] = output
	// 	return
	// }

	var userData models.User
	if err := json.Unmarshal(u.Ctx.Input.RequestBody, &userData); err != nil {
		u.Ctx.Output.SetStatus(400)
		output["error"] = "Data tidak dapat diproses"
		u.Data["json"] = output
		return
	}

	nama := strings.TrimSpace(userData.Nama)
	alamat := strings.TrimSpace(userData.Alamat)
	email := strings.TrimSpace(userData.Email)
	password := userData.Password

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

	hash, err := middlewares.HashPassword(password)
	if err != nil {
		u.Ctx.Output.SetStatus(404)
		output["error"] = "Password tidak dapat diproses"
		u.Data["json"] = output
		return
	}
	userData.Password = hash
	userData.Email = email

	/**
	Check existing first
	**/
	check := make(map[string]interface{})
	check["email"] = email
	doesExist, err := u.DoesUserExist(check)
	if err == nil && doesExist != nil {
		u.Ctx.Output.SetStatus(403)
		output["error"] = "User sudah terdaftar"
		u.Data["json"] = output
		return
	}

	userModel := new(models.UserModel)
	user, err := userModel.Create(&userData)
	if err != nil {
		u.Ctx.Output.SetStatus(404)
		output["error"] = err.Error()
		u.Data["json"] = output
		return
	}
	userMap := user.(map[string]interface{})

	delete(userMap, "password")

	output["object"] = map[string]interface{}{
		"user": userMap,
	}
	output["success"] = true
	u.Data["json"] = output

}
func (c *AuthController) DoesUserExist(params map[string]interface{}, excludeKey ...string) (interface{}, error) {
	if params["email"] == nil || params["email"].(string) == "" {
		return nil, errors.New("Missing required identifier(s)")
	}

	userModel := new(models.UserModel)
	obj, err := userModel.GetObjectByParams(params)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
