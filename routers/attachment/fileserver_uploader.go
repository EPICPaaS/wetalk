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

package attachment

import (
	"time"

	"github.com/astaxie/beego"

	"github.com/beego/wetalk/modules/attachment"
	"github.com/beego/wetalk/modules/models"
	"github.com/beego/wetalk/routers/base"
)

type FileServerUploadRouter struct {
	base.BaseRouter
}

func (this *FileServerUploadRouter) Post() {
	result := map[string]interface{}{
		"success": false,
	}

	defer func() {
		this.Data["json"] = &result
		this.ServeJson()
	}()

	// check permition
	if !this.User.IsActive {
		return
	}

	// get file object
	file, handler, err := this.Ctx.Request.FormFile("image")
	if err != nil {
		return
	}
	defer file.Close()

	t := time.Now()

	image := models.Image{}
	image.User = &this.User

	// get mime type
	mime := handler.Header.Get("Content-Type")

	// save and resize image
	fileUrl, err := attachment.SaveImageToFileServer(&image, file, mime, handler.Filename, t)
	if err != nil {
		beego.Error(err)
		return
	}

	result["link"] = "http://" + fileUrl
	result["success"] = true

}