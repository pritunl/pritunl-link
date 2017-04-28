package ipsec

import "html/template"

const (
	conf = `version 2.0

config setup
	protostack=netkey
	nat_traversal=yes
	virtual_private=
	oe=off

include /etc/ipsec.d/*.conf
`
	secrets = `include /etc/ipsec.d/*.secrets
`
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
	right={{.Right}}
	rightid=@{{.Right}}
	rightsubnet={{.RightSubnets}}
	auto=start
`
	secretsTemplateStr = `@{{.Left}} @{{.Right}}: PSK "{{.PreSharedKey}}"
`
)

var (
	confTemplate = template.Must(
		template.New("conf").Parse(confTemplateStr))
	secretsTemplate = template.Must(
		template.New("secrets").Parse(secretsTemplateStr))
)
