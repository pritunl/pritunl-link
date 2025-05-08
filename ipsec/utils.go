package ipsec

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/utils"
)

func GetWgIfaces() (ifacesSet set.Set, activeIfacesSet set.Set, err error) {
	ifacesSet = set.NewSet()
	activeIfacesSet = set.NewSet()

	output, err := utils.ExecOutput("", "wg", "show", "all", "dump")
	if err != nil {
		err = nil
	} else {
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) < 4 {
				continue
			}

			iface := fields[0]
			if !strings.HasPrefix(iface, "wgp") || len(iface) < 10 {
				continue
			}

			ifacesSet.Add(iface)
			activeIfacesSet.Add(iface)
		}
	}

	files, err := os.ReadDir(constants.WgDirPath)
	if err != nil {
		err = nil
	} else {
		for _, file := range files {
			filename := file.Name()
			if strings.HasPrefix(filename, "wgp") &&
				strings.HasSuffix(filename, ".conf") {

				ifaceName := strings.TrimSuffix(filename, ".conf")
				ifacesSet.Add(ifaceName)
			}
		}
	}

	return
}

func GetDirectSubnet() (network *net.IPNet, err error) {
	networkStr := config.Config.DirectSubnet
	if networkStr == "" {
		networkStr = defaultDirectNetwork
	}

	_, network, err = net.ParseCIDR(networkStr)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "ipsec: Failed to prase direct subnet"),
		}
		return
	}

	return
}

func GetDirectCidr() string {
	networkStr := config.Config.DirectSubnet
	if networkStr == "" {
		networkStr = defaultDirectNetwork
	}

	networkSpl := strings.Split(networkStr, "/")

	return networkSpl[len(networkSpl)-1]
}

func GetDirectServerIp() (ip net.IP, err error) {
	network, err := GetDirectSubnet()
	if err != nil {
		return
	}

	ip = network.IP
	utils.IncIpAddress(ip)

	return
}

func GetDirectClientIp() (ip net.IP, err error) {
	network, err := GetDirectSubnet()
	if err != nil {
		return
	}

	ip = network.IP
	utils.IncIpAddress(ip)
	utils.IncIpAddress(ip)

	return
}

func GetDirectMode() (mode string) {
	mode = config.Config.DirectMode
	if mode == "" {
		mode = defaultDirectMode
	}
	return
}

func Shutdown(connId string) {
	for i := 0; i < 5; i++ {
		_ = utils.Exec("", "ipsec", "down", connId)
		time.Sleep(50 * time.Millisecond)
	}
}

func PreSharedKeyToWg(psk string) string {
	hash := sha256.Sum256([]byte(psk))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func GetWgIface(id string) string {
	hash := md5.New()
	hash.Write([]byte(id))
	hashSum := base32.StdEncoding.EncodeToString(hash.Sum(nil))[:11]
	return fmt.Sprintf("wgp%s", strings.ToLower(hashSum))
}
