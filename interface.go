package webx

import "github.com/shestakovda/errx"

func NewRequest(url string) (Request, error) { return newRequestV1(url) }

type Request interface {
	Make(string, ...Option) (Response, error)
}

type Response interface {
	Body() []byte
	Status() int
}

type Option func(*options) error

var (
	ErrBadURL      = errx.New("Некорректное значение адреса")
	ErrBadOption   = errx.New("Некорректное значение аргумента")
	ErrBadRequest  = errx.New("Некорректные данные запроса")
	ErrBadResponse = errx.New("Некорректные данные ответа")
)
