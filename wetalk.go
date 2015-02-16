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

// An open source project for Gopher community.
package main

import (
	"github.com/astaxie/beego"
	"github.com/beego/social-auth"

	"github.com/astaxie/beego/orm"
	"github.com/beego/wetalk/routers"
	"github.com/beego/wetalk/routers/auth"
	"github.com/beego/wetalk/setting"
	_ "github.com/go-sql-driver/mysql"
	. "github.com/qiniu/api/conf"
)

// We have to call a initialize function manully
// because we use `bee bale` to pack static resources
// and we cannot make sure that which init() execute first.
func initialize() {
	setting.LoadConfig()

	//set logger
	if setting.IsProMode {
		beego.SetLogger("file", `{"filename":"logs/prod.log"}`)
		beego.SetLevel(beego.LevelInformational)
		beego.BeeLogger.DelLogger("console")
	} else {
		beego.SetLogger("file", `{"filename":"logs/dev.log"}`)
		beego.SetLevel(beego.LevelDebug)
		beego.BeeLogger.SetLogger("console", "")
	}
	beego.SetLogFuncCall(true)
	setting.SocialAuth = social.NewSocial("/login/", auth.SocialAuther)
	setting.SocialAuth.ConnectSuccessURL = "/settings/profile"
	setting.SocialAuth.ConnectFailedURL = "/settings/profile"
	setting.SocialAuth.ConnectRegisterURL = "/register/connect"
	setting.SocialAuth.LoginURL = "/login"

	//Qiniu
	ACCESS_KEY = setting.QiniuAccessKey
	SECRET_KEY = setting.QiniuSecurityKey
}

func main() {
	initialize()

	beego.Info("AppPath:", beego.AppPath)

	if setting.IsProMode {
		beego.Info("Product mode enabled")
	} else {
		beego.Info("Develment mode enabled")
	}
	beego.Info(beego.AppName, setting.APP_VER, setting.AppUrl)

	//initialize the routers
	routers.Initialize()
	if !setting.IsProMode {
		beego.SetStaticPath("/static_source", "static_source")
		beego.DirectoryIndex = true
	}

	if beego.RunMode == "dev" {
		//enable debug for orm
		orm.Debug = false
	}

	// For all unknown pages.
	beego.Run()
}
