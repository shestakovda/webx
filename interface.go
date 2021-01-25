package webx

import "github.com/shestakovda/errx"

const (
	HeaderXAPIKey       = "X-API-Key"
	HeaderContentEnc    = "Content-Transfer-Encoding"
	HeaderContentType   = "Content-Type"
	HeaderContentDisp   = "Content-Disposition"
	HeaderLastModified  = "Last-Modified"
	HeaderAuthorization = "Authorization"

	MimeXML     = "text/xml; charset=utf-8"
	MimeZIP     = "application/zip; application/octet-stream"
	MimeTGZ     = "application/tar+gzip; application/gzip; application/octet-stream"
	MimeJSON    = "application/json; charset=utf-8"
	MimeText    = "text/html; charset=utf-8"
	MimeUnknown = "application/octet-stream"
)

func NewRequest(baseURL string, args ...Option) (Request, error) { return newRequestV1(baseURL, args) }

type Request interface {
	Make(string, ...Option) (Response, error)
}

type Response interface {
	URL() string
	Code() int
	Body() []byte
	Text() string
	File() (*File, error)
	JSON(interface{}) error
	Error() error
}

type File struct {
	Name   string
	Mime   string
	Data   []byte
	Escape bool
}

type Option func(*options) error

var (
	ErrBadURL      = errx.New("Некорректное значение адреса")
	ErrBadBody     = errx.New("Некорректный состав тела запроса")
	ErrBadOption   = errx.New("Некорректное значение аргумента")
	ErrBadRequest  = errx.New("Некорректные данные запроса")
	ErrBadResponse = errx.New("Некорректные данные ответа")
	ErrResponse    = errx.New("Ошибка выполнения запроса")
)
