package ipsec

import (
	"html/template"
)

const (
	confTemplateStr = `conn {{.Id}}
	ikelifetime=8h
	keylife=1h
	rekeymargin=9m
	keyingtries=%forever
	authby=secret
	keyexchange=ikev2
	mobike=no
	dpddelay=10s
	dpdtimeout=30s
	dpdaction=restart
	left=%defaultroute
	leftid=@{{.Left}}
	leftsubnet={{.LeftSubnets}}
	leftikeport=600
	right={{.Right}}
	rightid=@{{.Right}}
	rightsubnet={{.RightSubnets}}
	rightikeport=600
	auto=start
`
	secretsTemplateStr = `@{{.Left}} @{{.Right}} : PSK "{{.PreSharedKey}}"
`
)

var (
	confTemplate = template.Must(
		template.New("conf").Parse(confTemplateStr))
	secretsTemplate = template.Must(
		template.New("secrets").Parse(secretsTemplateStr))
)
