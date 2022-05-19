package handlers

import (
	"beeapi/controllers"
	"beeapi/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/dgrijalva/jwt-go"
)

func init() {
}

func Jwt(ctx *context.Context) {
	ctx.Output.Header("Content-Type", "application/json")
	var uri string = ctx.Input.URI()
	if uri == "/v1/jwt" || uri == "/v1/auth/login" || uri == "/v1/auth/register" {
		return
	}

	if ctx.Input.Header("Authorization") == "" {
		ctx.Output.SetStatus(403)

		responseBody := controllers.Response{Code: http.StatusForbidden, Message: "Not Authorize", Data: nil}
		ctx.Output.Status = http.StatusInternalServerError
		resBytes, err := json.Marshal(responseBody)
		if err != nil {
			panic(err.Error())
		}

		ctx.Output.Body(resBytes)
		return
	}

	var tokenString string = ctx.Input.Header("Authorization")
	newToken := strings.Replace(tokenString, "Bearer ", "", 1)

	// fmt.Println(newToken)
	token, err := jwt.Parse(newToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(beego.AppConfig.String("JWTSECRET")), nil
	})

	if err != nil {
		ctx.Output.SetStatus(403)
		responseBody := models.APIResponse{Code: 403, Message: err.Error()}
		resBytes, err := json.Marshal(responseBody)
		ctx.Output.Body(resBytes)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("tooookkkeeeen", token, "toookeeeen")
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid && claims != nil {
		return
	} else {
		ctx.Output.SetStatus(403)
		resBody, err := json.Marshal(models.APIResponse{Code: 403, Message: ctx.Input.Header("Authorization")})
		ctx.Output.Body(resBody)
		if err != nil {
			panic(err)
		}
	}
}
