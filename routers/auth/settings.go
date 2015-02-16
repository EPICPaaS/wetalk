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
	"github.com/astaxie/beego"

	"github.com/beego/wetalk/modules/auth"
	"github.com/beego/wetalk/routers/base"
	"github.com/beego/wetalk/setting"
)

// SettingsRouter serves user settings.
type SettingsRouter struct {
	base.BaseRouter
}

func (this *SettingsRouter) ChangePassword() {
	this.Data["IsUserSettingPage"] = true
	this.TplNames = "settings/change_password.html"

	//need login
	if this.CheckLoginRedirect() {
		return
	}

	formPwd := auth.PasswordForm{}
	this.SetFormSets(&formPwd)
}

func (this *SettingsRouter) ChangePasswordSave() {
	this.Data["IsUserSettingPage"] = true
	this.TplNames = "settings/change_password.html"
	if this.CheckLoginRedirect() {
		return
	}

	pwdForm := auth.PasswordForm{User: &this.User}

	this.Data["Form"] = pwdForm

	if this.ValidFormSets(&pwdForm) {
		// verify success and save new password
		if err := auth.SaveNewPassword(&this.User, pwdForm.Password); err == nil {
			this.FlashRedirect("/settings/change/password", 302, "PasswordSave")
			return
		} else {
			beego.Error("ProfileSave: change-password", err)
		}
	}
}

func (this *SettingsRouter) AvatarSetting() {
	this.Data["IsUserSettingPage"] = true
	this.TplNames = "settings/user_avatar.html"
	//need login
	if this.CheckLoginRedirect() {
		return
	}

	form := auth.UserAvatarForm{}
	form.SetFromUser(&this.User)
	this.SetFormSets(&form)
}

func (this *SettingsRouter) AvatarSettingSave() {
	this.Data["IsUserSettingPage"] = true
	this.TplNames = "settings/user_avatar.html"
	//need login
	if this.CheckLoginRedirect() {
		return
	}
	avatarType, _ := this.GetInt("AvatarType")
	form := auth.UserAvatarForm{AvatarType: int(avatarType)}
	this.Data["Form"] = form

	if this.ValidFormSets(&form) {
		if err := auth.SaveAvatarType(&this.User, int(avatarType)); err == nil {
			this.FlashRedirect("/settings/avatar", 302, "AvatarSettingSave")
			return
		} else {
			beego.Error("ProfileSave: avatar-setting", err)
		}
	}
}

func (this *SettingsRouter) AvatarUpload() {
	this.Data["IsUserSettingPage"] = true
	this.TplNames = "settings/user_avatar.html"
	//need login and active
	if this.CheckLoginRedirect() {
		return
	}

	// get file object
	file, handler, err := this.Ctx.Request.FormFile("avatar")
	if err != nil {
		return
	}
	defer file.Close()
	mime := handler.Header.Get("Content-Type")
	if err := auth.UploadUserAvatarToQiniu(file, handler.Filename, mime, setting.QiniuAvatarBucket, &this.User); err != nil {
		return
	}

	userAvatarForm := auth.UserAvatarForm{}
	userAvatarForm.SetFromUser(&this.User)
	this.SetFormSets(&userAvatarForm)
	this.FlashRedirect("/settings/avatar", 302, "AvatarUploadSuccess")
}

// Profile implemented user profile settings page.
func (this *SettingsRouter) Profile() {
	this.Data["IsUserSettingPage"] = true
	this.TplNames = "settings/profile.html"

	// need login
	if this.CheckLoginRedirect() {
		return
	}

	form := auth.ProfileForm{Locale: this.Locale}

	form.SetFromUser(&this.User)
	this.SetFormSets(&form)
}

// ProfileSave implemented save user profile.
func (this *SettingsRouter) ProfileSave() {
	this.Data["IsUserSettingPage"] = true
	this.TplNames = "settings/profile.html"
	if this.CheckLoginRedirect() {
		return
	}

	action := this.GetString("action")
	if this.IsAjax() {
		switch action {
		case "send-verify-email":
			if this.User.IsActive {
				this.Data["json"] = false
			} else {
				auth.SendActiveMail(this.Locale, &this.User)
				this.Data["json"] = true
			}

			this.ServeJson()
			return
		}
		return
	}

	profileForm := auth.ProfileForm{Locale: this.Locale}
	profileForm.SetFromUser(&this.User)

	this.Data["Form"] = profileForm

	if this.ValidFormSets(&profileForm) {
		if err := profileForm.SaveUserProfile(&this.User); err != nil {
			beego.Error("ProfileSave: save-profile", err)
		}
		this.FlashRedirect("/settings/profile", 302, "ProfileSave")
		return
	}

}
