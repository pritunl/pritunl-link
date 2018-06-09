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
	esp=aes128-sha256-modp3072,aes256-sha256-modp3072,aes128-sha1-modp3072,aes128-sha256-modp4096,aes256-sha256-modp4096,aes128-sha1-modp4096,aes128-sha256-modp2048,aes256-sha256-modp2048,aes128-sha1-modp2048,aes128-sha256,aes256-sha256,aes128-sha1
	ike=aes128-sha256-modp3072,aes256-sha256-modp3072,aes128-sha1-modp3072,aes128-sha256-modp4096,aes256-sha256-modp4096,aes128-sha1-modp4096,aes128-sha256-modp2048,aes256-sha256-modp2048,aes128-sha1-modp2048
	mobike=no
	dpddelay=5s
	dpdtimeout=20s
	dpdaction=restart
	left=%defaultroute
	leftid={{.Left}}
	leftsubnet={{.LeftSubnets}}
	right={{.Right}}
	rightid={{.Right}}
	rightsubnet={{.RightSubnets}}
	auto=start
`
	secretsTemplateStr = `{{.Left}} {{.Right}} : PSK "{{.PreSharedKey}}"
`
)

var (
	confTemplate = template.Must(
		template.New("conf").Parse(confTemplateStr))
	secretsTemplate = template.Must(
		template.New("secrets").Parse(secretsTemplateStr))
)
