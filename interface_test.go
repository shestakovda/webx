package webx_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/webx"
	"github.com/stretchr/testify/suite"
)

func TestWebx(t *testing.T) {
	suite.Run(t, new(WebxSuite))
}

type WebxSuite struct {
	suite.Suite

	hdl http.HandlerFunc
	srv *httptest.Server
}

func (s *WebxSuite) SetupTest() {
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.hdl(w, r) }))
}

func (s *WebxSuite) TearDownTest() {
	s.srv.Close()
}

func (s *WebxSuite) TestBase() {
	const msg = "some test message"

	if _, err := webx.NewRequest("/base/"); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadURL))
	}

	// Формируем базовый запрос
	req, err := webx.NewRequest(s.srv.URL + "/base/")
	s.Require().NoError(err)

	// Запрос без всяких доп.параметров
	s.hdl = func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/base/test/", r.URL.String())
		w.Write([]byte(msg))
	}

	// Выполняем и получаем ответ
	res, err := req.Make("/test/")
	s.Require().NoError(err)

	// Базовое сравнение ответа
	s.Equal(s.srv.URL+"/base/test/", res.URL())
	s.Equal(http.StatusOK, res.Code())
	s.Equal([]byte(msg), res.Body())
	s.Equal(msg, res.Text())
}

func (s *WebxSuite) TestArgsHeaders() {
	// Формируем базовый запрос
	req, err := webx.NewRequest(
		s.srv.URL+"/base",
		webx.Arg("key1", "val1"),
		webx.Arg("key2", "val2"),
		webx.SetArg("key3", "val3"),
		webx.Header("key4", "val4"),
		webx.Header("key5", "val5"),
		webx.SetHeader("key6", "val6"),
	)
	s.Require().NoError(err)

	// Проверим get-параметры и заголовки
	s.hdl = func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodHead, r.Method)
		s.Equal("/base/test/?key1=val1&key2=val2&key2=val7&key3=val8", r.URL.String())
		s.Equal([]string{"val5", "val9"}, r.Header.Values("key5"))
		s.Equal([]string{"val10"}, r.Header.Values("key6"))
		s.Equal("val4", r.Header.Get("key4"))
		w.WriteHeader(http.StatusOK)
	}

	// Выполняем и получаем ответ
	res, err := req.Make(
		"test/",
		webx.HEAD(),
		webx.Arg("key2", "val7"),
		webx.SetArg("key3", "val8"),
		webx.Header("key5", "val9"),
		webx.SetHeader("key6", "val10"),
	)
	s.Require().NoError(err)

	// Базовое сравнение ответа
	s.Equal(http.StatusOK, res.Code())
	s.Empty(res.Body())
}

func (s *WebxSuite) TestJSON() {
	const msg = `{"ololo": "purpur"}`

	// Формируем базовый запрос
	req, err := webx.NewRequest(s.srv.URL + "/base/")
	s.Require().NoError(err)

	// Запрос без всяких доп.параметров
	s.hdl = func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/base/json/", r.URL.String())
		if data, err := ioutil.ReadAll(r.Body); s.NoError(err) {
			s.Equal(`{"ololo":"awful"}`, string(data))
		}
		w.Write([]byte(msg))
	}

	// Выполняем и получаем ответ
	res, err := req.Make("/json/", webx.POST(), webx.JSON(&dummy{"awful"}))
	s.Require().NoError(err)

	// Базовое сравнение ответа
	s.Equal(http.StatusOK, res.Code())
	s.Equal([]byte(msg), res.Body())
	s.Equal(msg, res.Text())

	dum := new(dummy)
	if err := res.JSON(dum); s.NoError(err) {
		s.Equal("purpur", dum.Ololo)
	}
}

func (s *WebxSuite) TestFormError() {
	const msg = `suck a lemon!`

	// Формируем базовый запрос
	req, err := webx.NewRequest(s.srv.URL + "/base/")
	s.Require().NoError(err)

	// Запрос без всяких доп.параметров
	s.hdl = func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPut, r.Method)
		s.Equal("/base/file/", r.URL.String())

		// Проверяем авторизацию
		if user, pass, ok := r.BasicAuth(); s.True(ok) {
			s.Equal("test", user)
			s.Equal("pass", pass)
		}

		if err := r.ParseMultipartForm(128); s.NoError(err) {
			s.Equal("message", r.Form.Get("text"))
			s.Equal(`{"ololo":"awful"}`, r.Form.Get("json"))
			if file, head, err := r.FormFile("file"); s.NoError(err) {
				s.Equal("f1", head.Filename)
				s.Equal("", head.Header.Get(webx.HeaderContentEnc))
				if data, err := ioutil.ReadAll(file); s.NoError(err) {
					s.Equal("text1", string(data))
				}
			}
			if file, head, err := r.FormFile("b64"); s.NoError(err) {
				s.Equal("f2", head.Filename)
				s.Equal("base64", head.Header.Get(webx.HeaderContentEnc))
				if data, err := ioutil.ReadAll(file); s.NoError(err) {
					s.Equal("eyJvbG9sbyI6ICJwdXJwdXIifQ==", string(data))
				}
			}
		}

		// Отправляем в ответ ошибку
		http.Error(w, msg, http.StatusForbidden)
	}

	// Выполняем и получаем ответ
	if res, err := req.Make(
		"/file/",
		webx.PUT(),
		webx.Auth("test", "pass"),
		webx.Client(http.DefaultClient),
		webx.FieldStr("text", "message"),
		webx.FieldJSON("json", &dummy{"awful"}),
		webx.FieldFile("file", webx.File{Name: "f1", Data: []byte("text1")}),
		webx.FieldFileAsBase64("b64", webx.File{Name: "f2", Data: []byte(`{"ololo": "purpur"}`)}),
	); s.Error(err) {
		s.True(errx.Is(err, errx.ErrForbidden))

		// Базовое сравнение ответа
		if s.NotNil(res) {
			s.Equal(http.StatusForbidden, res.Code())
			s.Equal([]byte(msg+"\n"), res.Body())
		}
	}
}

func (s *WebxSuite) TestFileResp() {
	const msg = `{"ololo": "purpur"}`
	const b64 = "eyJvbG9sbyI6ICJwdXJwdXIifQ=="

	// Формируем базовый запрос
	req, err := webx.NewRequest(
		s.srv.URL+"/base/",
		webx.Auth("test", "pass"),
		webx.Client(http.DefaultClient),
	)
	s.Require().NoError(err)

	// Запрос без всяких доп.параметров
	s.hdl = func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPatch, r.Method)
		s.Equal("/base/some/", r.URL.String())

		// Проверяем авторизацию
		if user, pass, ok := r.BasicAuth(); s.True(ok) {
			s.Equal("test", user)
			s.Equal("pass", pass)
		}

		if data, err := ioutil.ReadAll(r.Body); s.NoError(err) {
			s.Equal(msg, string(data))
			w.Header().Set(webx.HeaderContentDisp, `attachement; filename="some.json"`)
			w.Header().Set(webx.HeaderContentType, webx.MimeJSON)
			w.Header().Set(webx.HeaderContentEnc, "Base64")
			w.Write([]byte(b64))
		}
	}

	// Выполняем и получаем ответ
	res, err := req.Make(
		"/some/",
		webx.PATCH(),
		webx.Body(webx.MimeText, bytes.NewBufferString(msg)),
	)
	s.Require().NoError(err)

	// Базовое сравнение ответа
	s.Equal(http.StatusOK, res.Code())
	s.Equal([]byte(b64), res.Body())
	s.Equal(b64, res.Text())

	if file, err := res.File(); s.NoError(err) {
		s.Equal(webx.MimeJSON, file.Mime)
		s.Equal("some.json", file.Name)
		s.Equal(msg, string(file.Data))
	}
}

func (s *WebxSuite) TestOptions() {
	const uri = "http://example.com"

	if _, err := webx.NewRequest(uri, webx.Arg("", "")); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.SetArg("", "")); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.Auth("", "")); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.Body("", nil)); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.Field("", nil)); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.FieldStr("", "")); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.FieldJSON("", nil)); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.FieldFile("", webx.File{})); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.FieldFile("test", webx.File{})); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.FieldFileAsBase64("", webx.File{})); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.FieldFileAsBase64("test", webx.File{})); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.Client(nil)); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.Header("", "")); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.SetHeader("", "")); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}

	if _, err := webx.NewRequest(uri, webx.Method("")); s.Error(err) {
		s.True(errx.Is(err, webx.ErrBadOption))
	}
}

type dummy struct {
	Ololo string `json:"ololo"`
}
