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
	type=tunnel
	authby=secret
	left=%defaultroute
	leftid={{.Left}}
	leftnexthop=%defaultroute
	leftsubnets={{"{"}}{{.LeftSubnets}}{{"}"}}
	right={{.Right}}
	rightsubnets={{"{"}}{{.RightSubnets}}{{"}"}}
	pfs=yes
	auto=start
`
	secretsTemplateStr = `{{.Left}} {{.Right}}: PSK "{{.PreSharedKey}}"
`
)

var (
	confTemplate = template.Must(
		template.New("conf").Parse(confTemplateStr))
	secretsTemplate = template.Must(
		template.New("secrets").Parse(secretsTemplateStr))
)
