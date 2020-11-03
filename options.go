package webx

import "net/http"

func getOpts(args []Option) (o options) {
	o.client = http.DefaultClient
	o.method = http.MethodGet

	for i := range args {
		args[i](&o)
	}

	return o
}

type options struct {
	method string
	client *http.Client
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
