package controllers

import (
	"beeapi/controllers/middlewares"
	"beeapi/models"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/astaxie/beego"
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
	var user LoginRequestFormat
	json.Unmarshal(u.Ctx.Input.RequestBody, &user)
	res, err := models.Login(user.Email, user.Password)
	if err != nil {

		data := Response{Code: http.StatusInternalServerError, Message: "email not found", Data: nil}
		u.Ctx.Output.Status = http.StatusInternalServerError
		u.Data["json"] = data
	}
	if checked := middlewares.CheckPasswordHash(user.Password, res.Password); !checked {
		data := Response{Code: http.StatusInternalServerError, Message: "invalid password", Data: nil}
		u.Ctx.Output.Status = http.StatusInternalServerError
		u.Data["json"] = data
	}

	token, err := middlewares.GenerateToken(res)

	data := ResponseLogin{Code: http.StatusOK, Message: "invalid password", Data: LoginData{Nama: res.Nama, Email: res.Email, Token: token}}
	u.Ctx.Output.Status = http.StatusOK
	u.Data["json"] = data

	u.ServeJSON()
}

func (u *AuthController) Register() {

	var user models.User
	json.Unmarshal(u.Ctx.Input.RequestBody, &user)

	userUid := "user_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	user.Uid = userUid
	hashPassword, _ := middlewares.HashPassword(user.Password)
	user.Password = hashPassword
	uid, err := models.AddUser(&user)
	if err != nil {

		u.Data["json"] = map[string]string{"error": err.Error()}
		u.ServeJSON()
	}

	data := Response{Code: http.StatusOK, Message: "success register", Data: uid}
	u.Ctx.Output.Status = http.StatusOK
	u.Data["json"] = data
	u.ServeJSON()

}
