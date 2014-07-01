package auth

import (
	"github.com/EPICPaaS/wetalk/modules/auth"
	"github.com/EPICPaaS/wetalk/modules/models"
	"github.com/EPICPaaS/wetalk/routers/base"
)

type RegisterExtendRouter struct {
	base.BaseRouter
}

func (this *RegisterExtendRouter) Register() {
	token := this.GetString("privateToken")
	if len(token) == 0 || token != "epic20140627yx" {
		str := "fail"
		this.Data["json"] = &str
		this.ServeJson()
		return
	}
	UserName := this.GetString("userName")
	Email := this.GetString("email")
	Password := this.GetString("password")
	// Create new user.
	user := new(models.User)
	err := auth.RegisterUser(user, UserName, Email, Password)
	if err != nil {
		str := "fail"
		this.Data["json"] = &str
		this.ServeJson()
		return
	}
	str := "true"
	this.Data["json"] = &str
	this.ServeJson()
}
