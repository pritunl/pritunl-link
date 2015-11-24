package utils

import (
	"github.com/pritunl/pritunl-auth/constants"
	"github.com/dropbox/godropbox/errors"
	"os"
	"os/exec"
)

func Exec(dir, name string, args ...string) (err error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if dir != "" {
		cmd.Dir = dir
	}

	err = cmd.Run()
	if err != nil {
		err = &constants.ExecError{
			errors.Wrapf(err, "utils: Failed to exec %s %s", name, args),
		}
		return
	}

	return
}

func ExecSilent(dir, name string, args ...string) (err error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	_, err = cmd.CombinedOutput()
	if err != nil {
		err = &constants.ExecError{
			errors.Wrapf(err, "utils: Failed to exec %s %s", name, args),
		}
		return
	}

	return
}

func ExecOutput(dir, name string, args ...string) (output string, err error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	outputByt, err := cmd.CombinedOutput()
	if err != nil {
		err = &constants.ExecError{
			errors.Wrapf(err, "utils: Failed to exec %s %s", name, args),
		}
		return
	}
	output = string(outputByt)

	return
}
