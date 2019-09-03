//+build !darwin

package proxy_settings

import "github.com/pkg/errors"

func EnableProxy(host string) (disable func() error, err error) {
	return func() error {
		return nil
	}, errors.New("automatically enabling system proxy is not yet implemented on your OS")
}
