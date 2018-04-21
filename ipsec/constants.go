package ipsec

import (
	"html/template"
)

const (
	DirectGre    = "gre"
	DirectVxlan  = "vxlan"
	DirectPolicy = "policy"
	DirectIface  = "pritunl0"

	defaultDirectNetwork = "10.197.197.196/30"
	defaultDirectMode    = DirectGre
	confTemplateStr      = `conn {{.Id}}
	ikelifetime=8h
	keylife=1h
	rekeymargin=9m
	keyingtries=%forever
	authby=secret
	keyexchange=ikev2
	mobike=no
	dpddelay=5s
	dpdtimeout=20s
	dpdaction=restart
	left=%defaultroute
	leftid={{.LeftId}}
	leftsubnet={{.LeftSubnets}}
	right={{.Right}}
	rightid={{.RightId}}
	rightsubnet={{.RightSubnets}}
	auto=start
`
	secretsTemplateStr = `{{.LeftId}} {{.RightId}} : PSK "{{.PreSharedKey}}"
`
)

var (
	confTemplate = template.Must(
		template.New("conf").Parse(confTemplateStr))
	secretsTemplate = template.Must(
		template.New("secrets").Parse(secretsTemplateStr))
)
