package middlewares

import (
	"context"
	"errors"

	"time"

	"github.com/astaxie/beego"
	"github.com/golang-jwt/jwt"
)

func GenerateToken(u map[string]interface{}) (string, error) {
	if u["_id"].(string) == "" {
		return "cannot Generate token", errors.New("id == 0")
	}

	codes := jwt.MapClaims{
		"user_uid": u["_id"].(string),
		// "email":    u.Email,
		// "password": u.Password,
		// "roles": u.Roles,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"auth": true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, codes)
	// fmt.Println(token)
	return token.SignedString([]byte(beego.AppConfig.String("JWTSECRET")))
}
func ExtractTokenUserUid(e context.Context) string {
	// user := e.Get("user").(*jwt.Token) //convert to jwt token from interface
	// if user.Valid {
	// 	codes := user.Claims.(jwt.MapClaims)
	// 	id := codes["user_uid"].(string)
	// 	return id
	// }
	return ""
}

func ExtractRoles(e context.Context) bool {
	// user := e.Get("user").(*jwt.Token) //convert to jwt token from interface
	// if user.Valid {
	// 	codes := user.Claims.(jwt.MapClaims)
	// 	id := codes["roles"].(bool)
	// 	return id
	// }
	return false
}
