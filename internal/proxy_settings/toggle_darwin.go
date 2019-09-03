package proxy_settings

import (
	"bytes"
	"github.com/pkg/errors"
	"net/url"
	"os/exec"
	"strings"
)

const (
	locationName = "grpc-proxy (enabled)"
)

// Overview:
//
// * Create a temporary network location and switch to it
//
// * Enable HTTP+HTTPS proxy for that location
//
// * Return a func that switches back to the original location
// and deletes the temporary one
func EnableProxy(host string) (disable func() error, err error) {
	defer func() {
		if disable == nil {
			disable = func() error {
				return nil
			}
		}

		if err != nil {
			// function returned an error so call the disable/teardown function
			// (and then set it to a dummy so no problem if the caller also calls it)
			disableErr := disable()
			if disableErr != nil {
				err = errors.WithMessagef(err, "failed to disable system proxy: %v\n", err)
			}
			disable = func() error {
				return nil
			}
		}
	}()

	currentLoc, err := exec.Command("networksetup", "-getcurrentlocation").CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get current network location: %s", string(currentLoc))
	}

	if !bytes.HasPrefix(currentLoc, []byte(locationName)) {
		var out []byte
		out, err = exec.Command("networksetup", "-createlocation", locationName, "populate").CombinedOutput()
		if err != nil && !bytes.Contains(out, []byte("is already a network location name")) {
			// failed and not because the location already exists
			return nil, errors.Wrapf(err, "failed to create temporary network location: %s", out)
		}

		err = exec.Command("networksetup", "-switchtolocation", locationName).Run()
		if err != nil {
			return nil, errors.Wrap(err, "failed to switch to temporary network location")
		}

		disable = func() error {
			err := exec.Command("networksetup", "-switchtolocation", strings.Trim(string(currentLoc), "\n")).Run()
			if err != nil {
				return errors.Wrap(err, "failed to switch back to original network location")
			}

			err = exec.Command("networksetup", "-deletelocation", locationName).Run()
			if err != nil {
				return errors.Wrap(err, "failed to delete temporary network location")
			}

			return nil
		}
	}
	// now in a temporary network location called locationName
	// so we can modify it at will

	networkServicesList, err := exec.Command("networksetup", "-listallnetworkservices").CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list network services: %s", string(networkServicesList))
	}

	proxyHost := &url.URL{Host: host}
	for _, service := range bytes.Split(networkServicesList, []byte{'\n'})[1:] {
		if len(service) == 0 {
			// skip the empty one at the end due to the trailing new line
			continue
		}

		err = exec.Command("networksetup", "-setwebproxy", string(service), proxyHost.Hostname(), proxyHost.Port()).Run()
		if err != nil {
			return nil, errors.Wrap(err, "failed to enable HTTP proxy")
		}

		err = exec.Command("networksetup", "-setsecurewebproxy", string(service), proxyHost.Hostname(), proxyHost.Port()).Run()
		if err != nil {
			return nil, errors.Wrap(err, "failed to enable HTTPS proxy")
		}
	}

	return disable, nil
}
