package webx

import (
	"net/http"
	"net/url"
)

func newRequestV1(base string) (req *v1Request, err error) {
	req = new(v1Request)

	if req.base, err = url.ParseRequestURI(base); err != nil {
		return nil, ErrBadURL.WithReason(err)
	}

	return req, nil
}

type v1Request struct {
	base *url.URL
}

func (c v1Request) Make(ref string, args ...Option) (_ Response, err error) {
	var refp *url.URL
	var reqs *http.Request
	var resp *http.Response

	opts := getOpts(args)

	if refp, err = url.Parse(ref); err != nil {
		return nil, ErrBadURL.WithReason(err)
	}

	if reqs, err = http.NewRequest(opts.method, c.base.ResolveReference(refp).String(), nil); err != nil {
		return nil, ErrBadRequest.WithReason(err)
	}

	if resp, err = opts.client.Do(reqs); err != nil {
		return nil, ErrBadRequest.WithReason(err)
	}

	return newResponseV1(resp)
}
