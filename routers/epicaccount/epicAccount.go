package epicaccount

import (
	"github.com/EPICPaaS/wetalk/modules/auth"
	"github.com/astaxie/beego"
)

type EpicAccount struct {
	beego.Controller
}

func (this *EpicAccount) Login() {
	token := this.GetString("token")
	result := auth.VerifyToken(token)
	if !result.Succeed {
		this.Data["json"] = "failed"
	} else {
		this.Ctx.SetCookie("epic_user_token", token, "/")
		this.Ctx.ResponseWriter.Header().Add("P3P", "CP='IDC DSP COR ADM DEVi TAIi PSA PSD IVAi IVDi CONi HIS OUR IND CNT'")
		this.Data["json"] = "succeed"
	}
	this.ServeJson()
	return
}
