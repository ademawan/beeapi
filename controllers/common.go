package controllers

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ResponseLogin struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    LoginData `json:"data"`
}
type LoginData struct {
	Nama  string `json:"nama"`
	Email string `json:"email"`
	Token string `json:"token"`
}

//=============================

type LoginRequestFormat struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *Response) ResponseToUser(code int, message string, data interface{}) *Response {
	return &Response{Code: code, Message: message, Data: data}
}
