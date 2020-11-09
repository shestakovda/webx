package webx

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/shestakovda/errx"
)

var ErrMsgMustBeAbs = "Базовый URL должен быть абсолютным"

var defClient = &http.Client{
	Timeout: time.Minute,
}

func newRequestV1(base string, args []Option) (req *v1Request, err error) {
	req = new(v1Request)

	if req.opts, err = getOpts(args); err != nil {
		return nil, ErrBadRequest.WithReason(err)
	}

	if req.base, err = url.ParseRequestURI(base); err != nil {
		return nil, ErrBadURL.WithReason(err).WithDebug(errx.Debug{
			"URL": base,
		})
	}

	if !req.base.IsAbs() {
		return nil, ErrBadURL.WithDetail(ErrMsgMustBeAbs).WithDebug(errx.Debug{
			"URL": base,
		})
	}

	return req, nil
}

type v1Request struct {
	opts options
	base *url.URL
}

func (c v1Request) Make(ref string, args ...Option) (_ Response, err error) {
	var req *http.Request
	var body io.Reader
	var opts options

	if opts, err = getOpts(args); err != nil {
		return nil, ErrBadRequest.WithReason(err)
	}

	if body, err = opts.Body(); err != nil {
		return nil, ErrBadRequest.WithReason(err)
	}

	addr := strings.TrimRight(c.base.String(), "/") + "/" + strings.TrimLeft(strings.TrimSpace(ref), "/")

	if req, err = http.NewRequest(opts.method, addr, body); err != nil {
		return nil, ErrBadRequest.WithReason(err).WithDebug(errx.Debug{
			"URL":    addr,
			"Method": opts.method,
		})
	}

	if err = c.applyGetArgs(req, &opts); err != nil {
		return nil, ErrBadRequest.WithReason(err)
	}

	if err = c.applyHeaders(req, &opts); err != nil {
		return nil, ErrBadRequest.WithReason(err)
	}

	return c.do(req, &opts)
}

func (c v1Request) applyGetArgs(req *http.Request, opts *options) error {

	// Возможно, какие-то аргументы уже указаны в запросе
	args := req.URL.Query()

	// Сначала параметры из базового запроса

	for name, list := range c.opts.addget {
		args[name] = append(args[name], list...)
	}

	for name := range c.opts.setget {
		args.Set(name, c.opts.setget.Get(name))
	}

	// Затем параметры из основного запроса

	for name, list := range opts.addget {
		args[name] = append(args[name], list...)
	}

	for name := range opts.setget {
		args.Set(name, opts.setget.Get(name))
	}

	// Конвертируются обратно
	req.URL.RawQuery = args.Encode()
	return nil
}

func (c v1Request) applyHeaders(req *http.Request, opts *options) error {
	// Сначала устанавливаются заголовки базового запроса
	for name, list := range c.opts.addhead {
		req.Header[name] = append(req.Header[name], list...)
	}

	for name := range c.opts.sethead {
		req.Header.Set(name, c.opts.sethead.Get(name))
	}

	// Затем устанавливаются заголовки самого запроса
	for name, list := range opts.addhead {
		req.Header[name] = append(req.Header[name], list...)
	}

	for name := range opts.sethead {
		req.Header.Set(name, opts.sethead.Get(name))
	}

	// Если никто так и не поставил тип содержимого - ставим мы
	if req.Header.Get(HeaderContentType) == "" {
		req.Header.Set(HeaderContentType, MimeUnknown)
	}

	// Если этому запросу нужна авторизация - применяем её
	if opts.user != "" {
		req.SetBasicAuth(opts.user, opts.pass)
		return nil
	}

	// Если авторизация в базовом запросе - применяем её
	if c.opts.user != "" {
		req.SetBasicAuth(c.opts.user, c.opts.pass)
	}

	return nil
}

func (c v1Request) do(req *http.Request, opts *options) (_ *v1Response, err error) {
	var resp *http.Response
	var client *http.Client

	if opts.client != nil {
		// Если в самом запросе указан клиент, используем его
		client = opts.client
	} else if c.opts.client != nil {
		// Если в базовом запросе указан клиент, используем его
		client = c.opts.client
	} else {
		// Если нигде указан - используем умолчания
		client = defClient
	}

	if c.opts.debug || opts.debug {
		var dump []byte

		if dump, err = httputil.DumpRequestOut(req, true); err != nil {
			return nil, ErrBadRequest.WithReason(err)
		}

		glog.Errorf("webx.Request = %s", dump)
		glog.Flush()
	}

	if resp, err = client.Do(req); err != nil {
		return nil, ErrBadRequest.WithReason(err).WithDebug(errx.Debug{
			"URL":    req.URL.String(),
			"Method": req.Method,
			"Length": req.ContentLength,
		})
	}

	return newResponseV1(req, resp)
}
