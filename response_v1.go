package webx

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/shestakovda/errx"
)

func newResponseV1(req *http.Request, res *http.Response) (r *v1Response, err error) {
	r = &v1Response{
		base: req,
		head: res.Header,
		code: res.StatusCode,
	}

	if res.Body != nil {
		defer res.Body.Close()

		if r.body, err = ioutil.ReadAll(res.Body); err != nil {
			return r, ErrBadResponse.WithReason(err)
		}
	}

	return r, r.Error()
}

type v1Response struct {
	code int
	body []byte
	head http.Header
	base *http.Request
}

func (r v1Response) URL() string  { return r.base.URL.String() }
func (r v1Response) Code() int    { return r.code }
func (r v1Response) Body() []byte { return r.body }
func (r v1Response) Text() string { return string(r.body) }
func (r v1Response) File() (_ *File, err error) {
	var cdh map[string]string

	if disp := r.head.Get(HeaderContentDisp); disp != "" {
		if _, cdh, err = mime.ParseMediaType(disp); err != nil {
			return nil, ErrBadResponse.WithReason(err).WithDebug(errx.Debug{
				"Значение":  disp,
				"Заголовки": r.head,
			})
		}
	} else {
		cdh = map[string]string{"filename": path.Base(r.base.URL.String())}
	}

	if strings.EqualFold(r.head.Get(HeaderContentEnc), "base64") {
		var n int

		buf := make([]byte, base64.StdEncoding.DecodedLen(len(r.body)))
		if n, err = base64.StdEncoding.Decode(buf, r.body); err != nil {
			return nil, ErrBadResponse.WithReason(err).WithDebug(errx.Debug{
				"Значение":  r.body,
				"Заголовки": r.head,
			})
		}
		r.body = buf[:n]
	}

	return &File{
		Name: cdh["filename"],
		Mime: r.head.Get(HeaderContentType),
		Data: r.body,
	}, nil
}
func (r v1Response) JSON(item interface{}) (err error) {
	if err = json.Unmarshal(r.body, item); err != nil {
		return ErrBadResponse.WithReason(err).WithDebug(errx.Debug{
			"Ответ": string(r.body),
		})
	}

	return nil
}
func (r v1Response) Error() error {
	var err errx.Error

	switch r.code {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent, http.StatusNotModified:
		return nil
	case http.StatusNotFound:
		err = ErrResponse.WithReason(errx.ErrNotFound)
	case http.StatusForbidden:
		err = ErrResponse.WithReason(errx.ErrForbidden)
	case http.StatusUnauthorized:
		err = ErrResponse.WithReason(errx.ErrUnauthorized)
	case http.StatusBadRequest, http.StatusMethodNotAllowed:
		err = ErrResponse.WithReason(errx.ErrBadRequest)
	default:
		err = ErrResponse.WithReason(errx.ErrUnavailable)
	}

	return err.WithDebug(errx.Debug{
		"Код":   r.code,
		"URL":   r.base.URL.String(),
		"Ответ": string(r.body),
	})
}
