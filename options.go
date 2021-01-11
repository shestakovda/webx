package webx

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/shestakovda/errx"
)

var rxQuote = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escQuotes(s string) string { return rxQuote.Replace(s) }

func getOpts(args []Option) (o options, err error) {
	o.method = http.MethodGet
	o.addget = make(url.Values, 4)
	o.setget = make(url.Values, 4)
	o.addhead = make(http.Header, 4)
	o.sethead = make(http.Header, 4)
	o.form = make(map[string][]byte, 4)
	o.file = make(map[string][]*formFile, 4)

	for i := range args {
		if err = args[i](&o); err != nil {
			return
		}
	}

	return o, nil
}

type options struct {
	user string
	pass string
	body io.Reader
	form map[string][]byte
	file map[string][]*formFile

	debug   bool
	method  string
	addget  url.Values
	setget  url.Values
	addhead http.Header
	sethead http.Header
	client  *http.Client

	ctx context.Context
}

func (o *options) Body() (_ io.Reader, err error) {
	if o.method == http.MethodGet || o.method == http.MethodHead {
		return nil, nil
	}

	if o.body != nil {
		return o.body, nil
	}

	if err = o.makeForm(); err != nil {
		return
	}

	return o.body, nil
}

func (o *options) makeForm() (err error) {
	var flw io.Writer

	buf := new(bytes.Buffer)
	form := multipart.NewWriter(buf)

	for field, data := range o.form {
		if flw, err = form.CreateFormField(field); err != nil {
			return ErrBadBody.WithReason(err)
		}

		if _, err = flw.Write(data); err != nil {
			return ErrBadBody.WithReason(err)
		}
	}

	for field := range o.file {
		for i := range o.file[field] {
			if flw, err = form.CreatePart(o.file[field][i].Header); err != nil {
				return ErrBadBody.WithReason(err)
			}

			if _, err = flw.Write(o.file[field][i].Buffer); err != nil {
				return ErrBadBody.WithReason(err)
			}
		}
	}

	o.sethead.Set(HeaderContentType, form.FormDataContentType())

	if err = form.Close(); err != nil {
		return ErrBadBody.WithReason(err)
	}

	o.body = bytes.NewReader(buf.Bytes())
	return nil
}

func AppendArg(name, value string) Option {
	return func(o *options) error {
		if name == "" {
			return ErrBadOption.WithStack()
		}

		o.addget.Add(name, value)
		return nil
	}
}

func ReplaceArg(name, value string) Option {
	return func(o *options) error {
		if name == "" {
			return ErrBadOption.WithStack()
		}

		o.setget.Set(name, value)
		return nil
	}
}

func Auth(user, pass string) Option {
	return func(o *options) error {
		if user == "" {
			return ErrBadOption.WithStack()
		}

		o.user = user
		o.pass = pass
		return nil
	}
}

func Body(mime string, body io.Reader) Option {
	return func(o *options) error {
		if mime == "" {
			return ErrBadOption.WithStack()
		}

		o.body = body
		o.sethead.Set(HeaderContentType, mime)
		return nil
	}
}

func Files(files map[string][]*File) Option {
	return func(o *options) error {
		if len(files) == 0 {
			return ErrBadOption.WithStack()
		}

		for field := range files {
			for i := range files[field] {
				if files[field][i] == nil || files[field][i].Name == "" {
					return ErrBadOption.WithStack().WithDebug(errx.Debug{
						"index": i,
					})
				}

				o.file[field] = append(o.file[field], newFormFile(field, files[field][i], false))
			}
		}

		return nil
	}
}

func Field(name string, data []byte) Option {
	return func(o *options) error {
		if name == "" {
			return ErrBadOption.WithStack()
		}

		o.form[name] = data
		return nil
	}
}

func FieldStr(name string, data string) Option {
	return Field(name, []byte(data))
}

func FieldJSON(name string, data interface{}) Option {
	return func(o *options) (err error) {
		if name == "" {
			return ErrBadOption.WithStack()
		}

		if o.form[name], err = json.Marshal(data); err != nil {
			return ErrBadOption.WithReason(err)
		}

		return nil
	}
}

func FieldFile(field string, files ...*File) Option {
	return func(o *options) error {
		if field == "" || len(files) == 0 {
			return ErrBadOption.WithStack()
		}

		for i := range files {
			if files[i] == nil || files[i].Name == "" {
				return ErrBadOption.WithStack().WithDebug(errx.Debug{
					"index": i,
				})
			}

			o.file[field] = append(o.file[field], newFormFile(field, files[i], false))
		}

		return nil
	}
}

func FieldFileAsBase64(field string, files ...*File) Option {
	return func(o *options) error {
		if field == "" || len(files) == 0 {
			return ErrBadOption.WithStack()
		}

		for i := range files {
			if files[i] == nil || files[i].Name == "" {
				return ErrBadOption.WithStack().WithDebug(errx.Debug{
					"index": i,
				})
			}

			o.file[field] = append(o.file[field], newFormFile(field, files[i], true))
		}
		return nil
	}
}

func JSON(item interface{}) Option {
	return func(o *options) (err error) {
		var buf []byte

		if buf, err = json.Marshal(item); err != nil {
			return ErrBadOption.WithReason(err)
		}

		o.body = bytes.NewReader(buf)
		o.sethead.Set(HeaderContentType, MimeJSON)
		return nil
	}
}

func Client(c *http.Client) Option {
	return func(o *options) error {
		if c == nil {
			return ErrBadOption.WithStack()
		}

		o.client = c
		return nil
	}
}

func AppendHeader(name, value string) Option {
	return func(o *options) error {
		if name == "" {
			return ErrBadOption.WithStack()
		}

		o.addhead.Add(name, value)
		return nil
	}
}

func ReplaceHeader(name, value string) Option {
	return func(o *options) error {
		if name == "" {
			return ErrBadOption.WithStack()
		}

		o.sethead.Set(name, value)
		return nil
	}
}

func Method(m string) Option {
	return func(o *options) error {
		if m == "" {
			return ErrBadOption.WithStack()
		}

		o.method = m
		return nil
	}
}

func GET() Option    { return Method(http.MethodGet) }
func PUT() Option    { return Method(http.MethodPut) }
func HEAD() Option   { return Method(http.MethodHead) }
func POST() Option   { return Method(http.MethodPost) }
func PATCH() Option  { return Method(http.MethodPatch) }
func DELETE() Option { return Method(http.MethodDelete) }

func Debug() Option {
	return func(o *options) error {
		o.debug = true
		return nil
	}
}

func Context(ctx context.Context) Option {
	return func(o *options) error {
		o.ctx = ctx
		return nil
	}
}

func newFormFile(field string, file *File, as64 bool) *formFile {
	const tpl = `form-data; name="%s"; filename*="UTF-8''%s"`

	f := &formFile{
		Header: make(textproto.MIMEHeader),
	}

	f.Header.Set(HeaderContentDisp, fmt.Sprintf(tpl, escQuotes(field), url.PathEscape(file.Name)))

	if file.Mime == "" {
		f.Header.Set(HeaderContentType, MimeUnknown)
	} else {
		f.Header.Set(HeaderContentType, file.Mime)
	}

	if as64 {
		f.Buffer = make([]byte, base64.StdEncoding.EncodedLen(len(file.Data)))
		base64.StdEncoding.Encode(f.Buffer, file.Data)
		f.Header.Set(HeaderContentEnc, "base64")
	} else {
		f.Buffer = file.Data
	}

	return f
}

type formFile struct {
	Buffer []byte
	Header textproto.MIMEHeader
}
