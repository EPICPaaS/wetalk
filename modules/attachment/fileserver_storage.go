package attachment

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	goio "io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/beego/wetalk/setting"

	"github.com/beego/wetalk/modules/models"
)

func SaveImageToFileServer(m *models.Image, r goio.ReadSeeker, mime string, filename string, created time.Time) error {
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
			return fmt.Errorf("unsupport image format `%s`", filename)
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
		return err
	}

	m.Width = img.Bounds().Dx()
	m.Height = img.Bounds().Dy()
	m.Created = created

	//save to database
	if err := m.Insert(); err != nil || m.Id <= 0 {
		return err
	}

	m.Token = m.GetToken()
	if err := m.Update(); err != nil {
		return err
	}

	//reset reader pointer
	if _, err := r.Seek(0, 0); err != nil {
		return err
	}

	// 上传文件服务器
	request, err := newfileUploadRequest(setting.StorageServer+"/submit", map[string]string{}, filename, r)
	if nil != err {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if nil != err {
		return err
	} else {
		body := &bytes.Buffer{}
		_, err := body.ReadFrom(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()
		fmt.Println(resp.StatusCode)
		fmt.Println(resp.Header)

		fmt.Println(body)
	}

	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// Creates a new file upload http request with optional extra params
func newfileUploadRequest(uri string, params map[string]string, filename string, r goio.Reader) (*http.Request, error) {
	fileContents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
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

	return http.NewRequest("POST", uri, body)
}
