package webx

import (
	"bytes"
	"io"
	"net/http"
)

func newResponseV1(base *http.Response) (res *v1Response, err error) {
	res = &v1Response{
		body:   new(bytes.Buffer),
		status: base.StatusCode,
	}

	if base.Body != nil {
		defer base.Body.Close()

		if _, err = io.Copy(res.body, base.Body); err != nil {
			return nil, ErrBadResponse.WithReason(err)
		}
	}

	return res, nil
}

type v1Response struct {
	body   *bytes.Buffer
	status int
}

func (r v1Response) Body() []byte { return r.body.Bytes() }
func (r v1Response) Status() int  { return r.status }
