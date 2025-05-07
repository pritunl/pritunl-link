package ipsec

import (
	"text/template"
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
	ike={{.IkeCiphers}}
	esp={{.EspCiphers}}
	mobike=yes
	dpddelay=5s
	dpdtimeout=15s
	dpdaction={{.Action}}
	left=%defaultroute
	leftid={{.Left}}
	leftsubnet={{.LeftSubnets}}
	right={{.Right}}
	rightid={{.Right}}
	rightsubnet={{.RightSubnets}}
	auto=start
`
	espCiphers         = "aes128gcm128-x25519,aes128-sha256-curve25519,aes128-sha256-modp2048s256,aes128-sha256-ecp256,aes128-sha256-modp3072,aes192-sha384-modp2048s256,aes192-sha384-ecp384,aes192-sha384-curve25519,aes256-sha512-modp2048s256,aes256-sha512-ecp521,aes256-sha512-curve25519,aes128-sha256-modp4096,aes128-sha256-modp2048,aes128-sha256-modp1536,aes128-sha1-modp2048s256,aes128-sha1-ecp256,aes128-sha1-modp3072,aes128-sha1-curve25519,aes128-sha1-modp4096,aes128-sha1-modp3072,aes128-sha1-modp2048,aes128-sha1-modp1536"
	ikeCiphers         = "aes128-sha256-x25519,aes128-sha256-curve25519,aes128-sha256-modp2048s256,aes128-sha256-ecp256,aes128-sha256-modp3072,aes192-sha384-modp2048s256,aes192-sha384-ecp384,aes192-sha384-curve25519,aes256-sha512-modp2048s256,aes256-sha512-ecp521,aes256-sha512-curve25519,aes128-sha256-modp4096,aes128-sha256-modp2048,aes128-sha256-modp1536,aes128-sha1-modp2048s256,aes128-sha1-ecp256,aes128-sha1-modp3072,aes128-sha1-curve25519,aes128-sha1-modp4096,aes128-sha1-modp3072,aes128-sha1-modp2048,aes128-sha1-modp1536"
	secretsTemplateStr = `{{.Left}} {{.Right}} : PSK "{{.PreSharedKey}}"
`
	confWgTemplateStr = `[Interface]
PrivateKey = {{.WgPrivateKey}}
ListenPort = 8273
`
	confWgPeerTemplateStr = `
[Peer]
PublicKey = {{.WgPublicKey}}
PresharedKey = {{.WgPreSharedKey}}
AllowedIPs = {{.RightSubnets}}
Endpoint = {{.RightWg}}:8273
PersistentKeepalive = 25
`
)

var (
	confTemplate = template.Must(
		template.New("conf").Parse(confTemplateStr))
	confWgTemplate = template.Must(
		template.New("wg_conf").Parse(confWgTemplateStr))
	confWgPeerTemplate = template.Must(
		template.New("wg_peer").Parse(confWgPeerTemplateStr))
	secretsTemplate = template.Must(
		template.New("secrets").Parse(secretsTemplateStr))
)
