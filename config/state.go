package config

import (
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/utils"
)

var State = &StateData{}

type Link struct {
	WgPublicKey  string `json:"wg_public_key"`
	WgPrivateKey string `json:"wg_private_key"`
}

type StateData struct {
	lock   sync.Mutex      `json:"-"`
	loaded bool            `json:"-"`
	Links  map[string]Link `json:"links"`
}

func (s *StateData) GenerateKey(linkId string) (
	pubKey, privKey string, err error) {

	privateKey, err := utils.GeneratePrivateKey()
	if err != nil {
		return
	}
	publicKey := privateKey.PublicKey()

	pubKey = publicKey.String()
	privKey = privateKey.String()

	s.lock.Lock()
	if s.Links == nil {
		s.Links = map[string]Link{}
	}
	s.Links[linkId] = Link{
		WgPublicKey:  pubKey,
		WgPrivateKey: privKey,
	}
	s.lock.Unlock()

	err = s.Save()
	if err != nil {
		return
	}

	return
}

func (s *StateData) GetPublicKey(linkId string) (pubKey string, err error) {
	s.lock.Lock()
	pubKey = s.Links[linkId].WgPublicKey
	s.lock.Unlock()

	if pubKey == "" {
		pubKey, _, err = s.GenerateKey(linkId)
		if err != nil {
			return
		}
	}

	return
}

func (s *StateData) GetPrivateKey(linkId string) (privKey string, err error) {
	s.lock.Lock()
	privKey = s.Links[linkId].WgPrivateKey
	s.lock.Unlock()

	if privKey == "" {
		_, privKey, err = s.GenerateKey(linkId)
		if err != nil {
			return
		}
	}

	return
}

func (s *StateData) Save() (err error) {
	saveLock.Lock()
	defer saveLock.Unlock()

	if !s.loaded {
		err = &errortypes.WriteError{
			errors.New("config: State file has not been loaded"),
		}
		return
	}

	data, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "config: File marshal error"),
		}
		return
	}

	err = utils.ExistsMkdir(constants.VarDir, 0755)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(constants.StatePath, data, 0600)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "config: File write error"),
		}
		return
	}

	return
}
