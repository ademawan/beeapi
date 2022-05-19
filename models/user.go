package models

import (
	"errors"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
)

type User struct {
	Uid      string `orm:"pk;size(50)" form:"uid" json:"uid"`
	Nama     string `orm:"size(128)" form:"nama" json:"nama"`
	Alamat   string `orm:"size(128)" form:"alamat" json:"alamat"`
	Email    string `orm:"size(128);unique" form:"email" json:"email"`
	Password string `orm:"size(128)" form:"password" json:"-"`
}

func init() {
	orm.RegisterModel(new(User))
}

func AddUser(user *User) (string, error) {
	o := orm.NewOrm()
	res, err := o.Raw("insert into user (uid,nama,alamat,email,password)values(?,?,?,?,?)", user.Uid, user.Nama, user.Alamat, user.Email, user.Password).Exec()
	if err != nil {
		return "", errors.New("failed insert data")
	}
	fmt.Println(res.RowsAffected())

	// u.Id = "user_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	return user.Uid, nil
}

func GetUserById(uid string) (v *User, err error) {
	o := orm.NewOrm()
	v = &User{Uid: uid}
	if err = o.QueryTable(new(User)).Filter("uid", uid).RelatedSel().One(v); err == nil {
		return v, nil
	}
	return nil, err
}

func GetAllUser() ([]User, error) {

	var sql string
	var users []User
	qb, _ := orm.NewQueryBuilder("mysql")
	qb.Select("id,nama,alamat").From("user").OrderBy("nama").Asc()

	sql = qb.String()

	o := orm.NewOrm()

	num, err := o.Raw(sql).QueryRows(&users)
	fmt.Println(num)

	if err != nil {
		return []User{}, err
	}

	return users, nil
	// o := orm.NewOrm()
	// qs := o.QueryTable(new(User))
	// // query k=v
	// for k, v := range query {
	// 	// rewrite dot-notation to Object__Attribute
	// 	k = strings.Replace(k, ".", "__", -1)
	// 	qs = qs.Filter(k, v)
	// }
	// // order by:
	// var sortFields []string
	// if len(sortby) != 0 {
	// 	if len(sortby) == len(order) {
	// 		// 1) for each sort field, there is an associated order
	// 		for i, v := range sortby {
	// 			orderby := ""
	// 			if order[i] == "desc" {
	// 				orderby = "-" + v
	// 			} else if order[i] == "asc" {
	// 				orderby = v
	// 			} else {
	// 				return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
	// 			}
	// 			sortFields = append(sortFields, orderby)
	// 		}
	// 		qs = qs.OrderBy(sortFields...)
	// 	} else if len(sortby) != len(order) && len(order) == 1 {
	// 		// 2) there is exactly one order, all the sorted fields will be sorted by this order
	// 		for _, v := range sortby {
	// 			orderby := ""
	// 			if order[0] == "desc" {
	// 				orderby = "-" + v
	// 			} else if order[0] == "asc" {
	// 				orderby = v
	// 			} else {
	// 				return nil, errors.New("Error: Invalid order. Must be either [asc|desc]")
	// 			}
	// 			sortFields = append(sortFields, orderby)
	// 		}
	// 	} else if len(sortby) != len(order) && len(order) != 1 {
	// 		return nil, errors.New("Error: 'sortby', 'order' sizes mismatch or 'order' size is not 1")
	// 	}
	// } else {
	// 	if len(order) != 0 {
	// 		return nil, errors.New("Error: unused 'order' fields")
	// 	}
	// }

	// var l []User
	// qs = qs.OrderBy(sortFields...).RelatedSel()
	// if _, err = qs.Limit(limit, offset).All(&l, fields...); err == nil {
	// 	if len(fields) == 0 {
	// 		for _, v := range l {
	// 			ml = append(ml, v)
	// 		}
	// 	} else {
	// 		// trim unused fields
	// 		for _, v := range l {
	// 			m := make(map[string]interface{})
	// 			val := reflect.ValueOf(v)
	// 			for _, fname := range fields {
	// 				m[fname] = val.FieldByName(fname).Interface()
	// 			}
	// 			ml = append(ml, m)
	// 		}
	// 	}
	// 	return ml, nil
	// }
	// return nil, err
}

// UpdateUser updates User by Id and returns error if
// the record to be updated doesn't exist
func UpdateUserById(m *User) (User, error) {
	o := orm.NewOrm()
	v := User{Uid: m.Uid}
	// ascertain id exists in the database
	if err := o.Read(&v); err == nil {
		var num int64
		if num, err = o.Update(m); err == nil {
			fmt.Println("Number of records updated in database:", num)
		}
	}
	return v, nil
}

func DeleteUser(uid string) (err error) {
	o := orm.NewOrm()
	v := User{Uid: uid}
	if err = o.Read(&v); err == nil {
		var num int64
		if num, err = o.Delete(&User{Uid: uid}); err == nil {
			fmt.Println("Number of records deleted in database:", num)
		}
	}
	return
}
func Login(email, password string) (*User, error) {
	o := orm.NewOrm()
	v := &User{Email: email}
	if err := o.QueryTable(new(User)).Filter("email", email).RelatedSel().One(v); err != nil {
		return &User{}, errors.New("email not found")
	}

	return v, nil
}
