package attachment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	goio "io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/beego/wetalk/setting"

	"github.com/beego/wetalk/modules/models"
)

func SaveImageToFileServer(m *models.Image, r goio.ReadSeeker, mime string, filename string, created time.Time) (string, error) {
	var ext string

	// check image mime type
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
			return "", fmt.Errorf("unsupport image format `%s`", filename)
		}
	}

	// decode image
	var img image.Image
	var err error
	switch ext {
	case ".jpg":
		m.Ext = 1
		img, err = jpeg.Decode(r)
	case ".png":
		m.Ext = 2
		img, err = png.Decode(r)
	case ".gif":
		m.Ext = 3
		img, err = gif.Decode(r)
	}

	if err != nil {
		return "", err
	}

	m.Width = img.Bounds().Dx()
	m.Height = img.Bounds().Dy()
	m.Created = created

	//save to database
	if err := m.Insert(); err != nil || m.Id <= 0 {
		return "", err
	}

	m.Token = m.GetToken()
	if err := m.Update(); err != nil {
		return "", err
	}

	//reset reader pointer
	if _, err := r.Seek(0, 0); err != nil {
		return "", err
	}

	// 上传文件服务器
	request, err := newfileUploadRequest(setting.StorageServer+"/submit", map[string]string{}, filename, r)
	if nil != err {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if nil != err {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	//		fmt.Println(resp.StatusCode)
	//		fmt.Println(resp.Header)

	//		fmt.Println(body)

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)

	if nil != err {
		return "", err
	}

	return data["fileUrl"].(string), nil
}

// Creates a new file upload http request with optional extra params
func newfileUploadRequest(uri string, params map[string]string, filename string, r goio.Reader) (*http.Request, error) {
	fileContents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)

	if err != nil {
		return nil, err
	}
	part.Write(fileContents)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	ret, err := http.NewRequest("POST", uri, &body)
	ret.Header.Set("Content-Type", writer.FormDataContentType())

	return ret, err
}
