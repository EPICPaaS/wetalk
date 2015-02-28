package epicaccount

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/beego/wetalk/modules/auth"
	"github.com/beego/wetalk/modules/models"
	"strconv"
)

type EpicAccount struct {
	beego.Controller
}

func (this *EpicAccount) Login() {
	token := this.GetString("token")
	result := auth.VerifyToken(token)

	fmt.Println(result.Username)
	fmt.Println(result.Userid)
	fmt.Println(result.Email)

	if !result.Succeed {
		this.Data["json"] = "failed"
	} else {
		//query weather this user exist in wetalk db
		userNew := models.User{}
		userid, err := strconv.Atoi(result.Userid)
		if err != nil {
			fmt.Println("用户Id转换出错了:" + err.Error())
		}

		//fmt.Println(result.Userid)

		userNew.Id = userid
		err = userNew.Read("Id")
		if err == nil {
			fmt.Println("获取用户信息失败 - nil")
		} else {
			fmt.Println("找不到用户 userId - " + result.Userid)
			userNew.UserName = result.Username
			userNew.Email = result.Email
			userNew.IsActive = true
			userNew.NickName = result.Username
			userNew.AvatarType = 2
			userNew.AvatarKey = "/static/img/default_avator.png"
			err = userNew.Insert()

			if err != nil {
				fmt.Println("插入用户信息出错 -" + err.Error())
			}

		}

		this.Ctx.SetCookie("epic_user_token", token, "/")
		this.Ctx.ResponseWriter.Header().Add("P3P", "CP='IDC DSP COR ADM DEVi TAIi PSA PSD IVAi IVDi CONi HIS OUR IND CNT'")
		this.Data["json"] = "succeed"
	}
	this.ServeJson()
	return
}
