package utils

import (
	"net/http"
	"net/url"
)

type Params struct {
	values url.Values
}

func (p *Params) GetByName(name string) (val string) {
	valList := p.values[name]

	if len(valList) > 0 {
		val = valList[0]
	}

	return
}

func ParseParams(req *http.Request) (parms *Params) {
	parms = &Params{
		values: req.URL.Query(),
	}
	return
}
