// Copyright 2013 wetalk authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package auth

import (
	"encoding/hex"
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/astaxie/beego/context"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/session"
	"github.com/beego/i18n"

	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beego/wetalk/modules/models"
	"github.com/beego/wetalk/modules/utils"
	"github.com/beego/wetalk/setting"
	qio "github.com/qiniu/api/io"
)

// CanRegistered checks if the username or e-mail is available.
func CanRegistered(userName string, email string) (bool, bool, error) {
	cond := orm.NewCondition()
	cond = cond.Or("UserName", userName).Or("Email", email)

	var maps []orm.Params
	o := orm.NewOrm()
	n, err := o.QueryTable("user").SetCond(cond).Values(&maps, "UserName", "Email")
	if err != nil {
		return false, false, err
	}

	e1 := true
	e2 := true

	if n > 0 {
		for _, m := range maps {
			if e1 && orm.ToStr(m["UserName"]) == userName {
				e1 = false
			}
			if e2 && orm.ToStr(m["Email"]) == email {
				e2 = false
			}
		}
	}

	return e1, e2, nil
}

// check if exist user by username or email
func HasUser(user *models.User, username string) bool {
	var err error
	qs := orm.NewOrm()
	if strings.IndexRune(username, '@') == -1 {
		user.UserName = username
		err = qs.Read(user, "UserName")
	} else {
		user.Email = username
		err = qs.Read(user, "Email")
	}
	if err == nil {
		return true
	}
	return false
}

// register create user
func RegisterUser(user *models.User, username, email, password string, locale i18n.Locale) error {
	// use random salt encode password
	salt := models.GetUserSalt()
	pwd := utils.EncodePassword(password, salt)

	user.UserName = strings.ToLower(username)
	user.Email = strings.ToLower(email)

	// save salt and encode password, use $ as split char
	user.Password = fmt.Sprintf("%s$%s", salt, pwd)

	// save md5 email value for gravatar
	user.GrEmail = utils.EncodeMd5(user.Email)

	// Use username as default nickname.
	user.NickName = user.UserName

	//set default language
	if locale.Lang == "en-US" {
		user.Lang = setting.LangEnUS
	} else {
		user.Lang = setting.LangZhCN
	}

	//set default avatar
	user.AvatarType = setting.AvatarTypeGravatar
	return user.Insert()
}

// set a new password to user
func SaveNewPassword(user *models.User, password string) error {
	salt := models.GetUserSalt()
	user.Password = fmt.Sprintf("%s$%s", salt, utils.EncodePassword(password, salt))
	return user.Update("Password", "Rands", "Updated")
}

//set a new avatar type to user
func SaveAvatarType(user *models.User, avatarType int) error {
	user.AvatarType = avatarType
	return user.Update("AvatarType", "Updated")
}

// get login redirect url from cookie
func GetLoginRedirect(ctx *context.Context) string {
	loginRedirect := strings.TrimSpace(ctx.GetCookie("login_to"))
	if utils.IsMatchHost(loginRedirect) == false {
		loginRedirect = "/"
	} else {
		ctx.SetCookie("login_to", "", -1, "/")
	}
	return loginRedirect
}

// login user
func LoginUser(user *models.User, ctx *context.Context, remember bool) {
	// werid way of beego session regenerate id...
	ctx.Input.CruSession.SessionRelease(ctx.ResponseWriter)
	ctx.Input.CruSession = beego.GlobalSessions.SessionRegenerateId(ctx.ResponseWriter, ctx.Request)
	ctx.Input.CruSession.Set("auth_user_id", user.Id)

	if remember {
		WriteRememberCookie(user, ctx)
	}
}

func WriteRememberCookie(user *models.User, ctx *context.Context) {
	secret := utils.EncodeMd5(user.Rands + user.Password)
	days := 86400 * setting.LoginRememberDays
	ctx.SetCookie(setting.CookieUserName, user.UserName, days)
	ctx.SetSecureCookie(secret, setting.CookieRememberName, user.UserName, days)
}

func DeleteRememberCookie(ctx *context.Context) {
	ctx.SetCookie(setting.CookieUserName, "", -1)
	ctx.SetCookie(setting.CookieRememberName, "", -1)
}

func LoginUserFromRememberCookie(user *models.User, ctx *context.Context) (success bool) {
	userName := ctx.GetCookie(setting.CookieUserName)
	if len(userName) == 0 {
		return false
	}

	defer func() {
		if !success {
			DeleteRememberCookie(ctx)
		}
	}()

	user.UserName = userName
	if err := user.Read("UserName"); err != nil {
		return false
	}

	secret := utils.EncodeMd5(user.Rands + user.Password)
	value, _ := ctx.GetSecureCookie(secret, setting.CookieRememberName)
	if value != userName {
		return false
	}

	LoginUser(user, ctx, true)

	return true
}

// logout user
func LogoutUser(ctx *context.Context) {
	DeleteRememberCookie(ctx)
	ctx.Input.CruSession.Delete("auth_user_id")
	ctx.Input.CruSession.Flush()
	beego.GlobalSessions.SessionDestroy(ctx.ResponseWriter, ctx.Request)
}

func GetUserIdFromSession(sess session.SessionStore) int {
	if id, ok := sess.Get("auth_user_id").(int); ok && id > 0 {
		return id
	}
	return 0
}

// get user if key exist in session
func GetUserFromSession(user *models.User, sess session.SessionStore) bool {
	id := GetUserIdFromSession(sess)
	if id > 0 {
		u := models.User{Id: id}
		if u.Read() == nil {
			*user = u
			return true
		}
	}

	return false
}

// verify username/email and password
func VerifyUser(user *models.User, username, password string) (success bool) {
	// search user by username or email
	if HasUser(user, username) == false {
		return
	}

	if VerifyPassword(password, user.Password) {
		// success
		success = true

		// re-save discuz password
		if len(user.Password) == 39 {
			if err := SaveNewPassword(user, password); err != nil {
				beego.Error("SaveNewPassword err: ", err.Error())
			}
		}
	}
	return
}

// compare raw password and encoded password
func VerifyPassword(rawPwd, encodedPwd string) bool {

	// for discuz accounts
	if len(encodedPwd) == 39 {
		salt := encodedPwd[:6]
		encoded := encodedPwd[7:]
		return encoded == utils.EncodeMd5(utils.EncodeMd5(rawPwd)+salt)
	}

	// split
	var salt, encoded string
	if len(encodedPwd) > 11 {
		salt = encodedPwd[:10]
		encoded = encodedPwd[11:]
	}

	return utils.EncodePassword(rawPwd, salt) == encoded
}

// get user by erify code
func getVerifyUser(user *models.User, code string) bool {
	if len(code) <= utils.TimeLimitCodeLength {
		return false
	}

	// use tail hex username query user
	hexStr := code[utils.TimeLimitCodeLength:]
	if b, err := hex.DecodeString(hexStr); err == nil {
		user.UserName = string(b)
		if user.Read("UserName") == nil {
			return true
		}
	}

	return false
}

// verify active code when active account
func VerifyUserActiveCode(user *models.User, code string) bool {
	minutes := setting.ActiveCodeLives

	if getVerifyUser(user, code) {
		// time limit code
		prefix := code[:utils.TimeLimitCodeLength]
		data := utils.ToStr(user.Id) + user.Email + user.UserName + user.Password + user.Rands

		return utils.VerifyTimeLimitCode(data, minutes, prefix)
	}

	return false
}

// create a time limit code for user active
func CreateUserActiveCode(user *models.User, startInf interface{}) string {
	minutes := setting.ActiveCodeLives
	data := utils.ToStr(user.Id) + user.Email + user.UserName + user.Password + user.Rands
	code := utils.CreateTimeLimitCode(data, minutes, startInf)

	// add tail hex username
	code += hex.EncodeToString([]byte(user.UserName))
	return code
}

// verify code when reset password
func VerifyUserResetPwdCode(user *models.User, code string) bool {
	minutes := setting.ResetPwdCodeLives

	if getVerifyUser(user, code) {
		// time limit code
		prefix := code[:utils.TimeLimitCodeLength]
		data := utils.ToStr(user.Id) + user.Email + user.UserName + user.Password + user.Rands + user.Updated.String()

		return utils.VerifyTimeLimitCode(data, minutes, prefix)
	}

	return false
}

// create a time limit code for user reset password
func CreateUserResetPwdCode(user *models.User, startInf interface{}) string {
	minutes := setting.ResetPwdCodeLives
	data := utils.ToStr(user.Id) + user.Email + user.UserName + user.Password + user.Rands + user.Updated.String()
	code := utils.CreateTimeLimitCode(data, minutes, startInf)

	// add tail hex username
	code += hex.EncodeToString([]byte(user.UserName))
	return code
}

//upload user avatar
func UploadUserAvatarToQiniu(r io.ReadSeeker, filename string, mime string, bucketName string, user *models.User) error {
	var ext string

	// test image mime type
	switch mime {
	case "image/jpeg":
		ext = ".jpg"

	case "image/png":
		ext = ".png"

	case "image/gif":
		ext = ".gif"

	default:
		ext = filepath.Ext(filename)
		switch ext {
		case ".jpg", ".png", ".gif":
		default:
			return fmt.Errorf("unsupport image format `%s`", filename)
		}
	}

	// decode image
	var err error
	switch ext {
	case ".jpg":
		_, err = jpeg.Decode(r)
	case ".png":
		_, err = png.Decode(r)
	case ".gif":
		_, err = gif.Decode(r)
	}

	if err != nil {
		return err
	}

	//reset reader pointer
	if _, err := r.Seek(0, 0); err != nil {
		return err
	}
	var data []byte
	if data, err = ioutil.ReadAll(r); err != nil {
		return err
	}

	if len(data) > setting.AvatarImageMaxLength {
		return fmt.Errorf("avatar image size too large", filename)
	}

	//reset reader pointer again
	if _, err := r.Seek(0, 0); err != nil {
		return err
	}

	//save to qiniu
	var uptoken = utils.GetQiniuUptoken(bucketName)
	var putRet qio.PutRet
	var putExtra = &qio.PutExtra{
		MimeType: mime,
	}

	err = qio.PutWithoutKey(nil, &putRet, uptoken, r, putExtra)
	if err != nil {
		return err
	}

	//update user
	user.AvatarKey = putRet.Key
	if err := user.Update("AvatarKey", "Updated"); err != nil {
		return err
	}
	return nil
}

type VerifyTokenResult struct {
	Succeed  bool   `json:"succeed"`
	Userid   string `json:"userid"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

//判断用户是否登陆获取用户信息
func GetUserFromCookie(user *models.User, ctx *context.Context) bool {

	token := ctx.GetCookie("epic_user_token")

	if len(token) == 0 {
		return false
	}
	result := VerifyToken(token)
	if !result.Succeed {
		return false
	}
	userNew := models.User{}
	userid, err := strconv.Atoi(result.Userid)
	if err != nil {
		fmt.Println("用户Id转换出错了:" + err.Error())
	}

	userNew.Id = userid
	err = userNew.Read("Id")
	if err == nil {
		*user = userNew
		return true
	} else {
		fmt.Println("获取用户信息失败 -" + err.Error())
	}
	return false

}

func VerifyToken(token string) VerifyTokenResult {
	result := VerifyTokenResult{}
	url := "http://" + setting.AccountCenterUrl + "/verify_token?token=" + token
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		fmt.Println("token校验失败")
		return result
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	json.Unmarshal(body, &result)
	return result
}

func DestroyToken(token string) bool {
	url := "http://" + setting.AccountCenterUrl + "/logout?token=" + token
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		fmt.Println("token销毁失败")
		return false
	} else {
		return true
	}
}
