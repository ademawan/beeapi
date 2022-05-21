package models

import (
	"errors"
	"strings"

	driver "github.com/arangodb/go-driver"
	"github.com/beego/beego/v2/client/orm"
	"github.com/davecgh/go-spew/spew"
)

type (
	UserModel struct {
		Limit   int
		Offset  int
		SortBy  string
		SortDir string
	}
	User struct {
		ID       string `json:"_id,omitempty"`
		Key      string `json:"_key,omitempty"`
		Nama     string `form:"nama" json:"nama"`
		Alamat   string `form:"alamat" json:"alamat"`
		Email    string `form:"email" json:"email"`
		Password string `form:"password" json:"password"`
	}
)

func init() {
	orm.RegisterModel(new(User))
}

func (m *UserModel) Create(data *User) (interface{}, error) {
	dbHandler := new(DbHandler)

	return dbHandler.SaveDocument("user", data)
}

//====================divide get==================
func (m *UserModel) GetObject(primaryKey string) (interface{}, error) {
	if primaryKey == "" {
		return nil, errors.New("Data id hilang")
	}
	dbHandler := new(DbHandler)

	return dbHandler.GetObject("user", primaryKey)
}

//=================get datatables===================
func (m *UserModel) GetDatatables(Qtab map[string]interface{}) ([]interface{}, int64, error) {
	bindVars := make(map[string]interface{})
	url := Qtab["url"].(map[string]interface{})
	query := "FOR doc IN user "
	//FOR doc IN user
	//LIMIT 0,10
	var selectStr string
	for k, v := range Qtab["columns"].([]string) {
		if k != 0 {
			selectStr += ","
		}
		selectStr += v + ":doc." + v
	}
	var whereStr string = "FILTER "
	var sortStr string = "SORT "

	search_len := len(url["search"].(string))
	colOffset := url["start"].(int)

	var limit string
	limit += "LIMIT @offset, @lenght "
	if search_len > 0 {
		for k, v := range Qtab["searchfilter"].([]string) {
			if k != 0 {
				whereStr += " OR "
			}
			whereStr += v + " LIKE " + "\"%" + url["search"].(string) + "%\"" //like
		}
		if url["order_dir"] == "asc" {
			query += whereStr
			query += sortStr + "doc.nama ASC "
			query += limit

		} else {
			query += whereStr
			query += sortStr + "doc.nama DESC "
			query += limit
		}
	} else {
		if url["order_dir"] == "asc" {
			query += sortStr + "doc.nama ASC "
			query += limit
		} else {
			query += sortStr + "doc.nama DESC "
			query += limit
		}
	}

	query += "RETURN {id:doc._id,alamat:doc.alamat,email:doc.email}"

	dbHandler := new(DbHandler)
	bindVars["offset"] = colOffset
	bindVars["lenght"] = url["lenght"].(int)
	dbHandler.BindVars = bindVars

	return dbHandler.GetCollectionByQueryWithCount(query)
}

//=============get object by params===================

func (m *UserModel) GetObjectByParams(params map[string]interface{}, excludeKey ...string) (interface{}, error) {
	if params == nil {
		return nil, errors.New("Data params hilang")
	}
	bindVars := make(map[string]interface{})
	query := "FOR doc IN user"
	var filters []string
	if params["email"] != nil && params["email"].(string) != "" {
		filters = append(filters, "(LIKE(@email, doc.email, true))")
		bindVars["email"] = params["email"].(string)
	}

	var andCondition []string
	if len(excludeKey) > 0 && excludeKey[0] != "" {
		andCondition = append(andCondition, "@userId != doc._id")
		bindVars["userId"] = "user/" + excludeKey[0]
	}
	if len(filters) > 0 {
		query += " FILTER " + strings.Join(filters, " OR ")
		if len(andCondition) > 0 {
			query += " AND " + strings.Join(andCondition, " AND ")
		}
	}
	query += " LIMIT 1 RETURN doc"

	dbHandler := new(DbHandler)
	dbHandler.BindVars = bindVars
	obj, err := dbHandler.GetObjectByQuery(query)
	if driver.IsNoMoreDocuments(err) {
		return nil, nil
	} else if err != nil {
		ZapLogger.Error(err.Error())
		var strDebug string
		strDebug = spew.Sdump(query)
		ZapLogger.Info(`query: ` + strDebug)
		strDebug = spew.Sdump(bindVars)
		ZapLogger.Info(`bindVars: ` + strDebug)
		return nil, err
	}

	return obj, nil
}

//================divide update=================

func (m *UserModel) Update(primaryKey string, data interface{}) (interface{}, error) {
	if primaryKey == "" {
		return nil, errors.New("Data id hilang")
	}
	if data == nil {
		return nil, errors.New("Data hilang")
	}

	// TODO check controller
	// data["migration_timestamp"] = MigrationTimestamp()

	user, err := m.GetObject(primaryKey)
	if err != nil {
		return nil, err
	}
	userMap := user.(map[string]interface{})
	dataMap := data.(map[string]interface{})
	for k, v := range dataMap {
		if k == "_id" || k == "_key" || k == "_rev" {
			continue
		}
		userMap[k] = v
	}
	dbHandler := new(DbHandler)
	saved, err := dbHandler.UpdateObject("user", primaryKey, userMap)
	if err != nil {
		return nil, err
	}

	return saved, nil
}

func (m *UserModel) Delete(primaryKey string) (interface{}, error) {
	if primaryKey == "" {
		return nil, errors.New("primaryKey hilang")
	}

	dbHandler := new(DbHandler)

	return dbHandler.RemoveObject("user", primaryKey)
}

func (m *UserModel) Login(email string) (interface{}, error) {
	dbHandler := new(DbHandler)

	return dbHandler.GetObjectByField("user", "email", email)
}
