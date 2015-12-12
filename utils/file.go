package utils

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-auth/errortypes"
	"io/ioutil"
	"os"
)

func Read(path string) (data string, err error) {
	dataByt, err := ioutil.ReadFile(path)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrapf(err, "utils: Failed to read '%s'", path),
		}
		return
	}
	data = string(dataByt)

	return
}

func Exists(path string) (exists bool, err error) {
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		err = nil
	} else if err != nil {
		err = errortypes.ReadError{
			errors.Wrapf(err, "utils: Failed to stat file '%s'", path),
		}
		return
	} else {
		exists = true
	}

	return
}

func Create(path string) (file *os.File, err error) {
	file, err = os.Create(path)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrapf(err, "utils: Failed to create '%s'", path),
		}
		return
	}

	return
}

func MkdirAll(path string) (err error) {
	err = os.MkdirAll(path, 0755)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrapf(err, "utils: Failed to create '%s'", path),
		}
		return
	}

	return
}

func Write(path string, data string) (err error) {
	file, err := Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = file.WriteString(data)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrapf(err, "utils: Failed to write to file '%s'", path),
		}
		return
	}

	return
}
