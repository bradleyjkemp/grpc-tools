package detectcert

import (
	"io/ioutil"
	"os"
	"strings"
)

var (
	keySuffix = "-key.pem"
	certSuffix = ".pem"
)

// Detect finds files in the current directory that look like
// mkcert-generated key-certificate pairs
func Detect() (cert string, key string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	files, err := ioutil.ReadDir(wd)
	if err != nil {
		return "", "", err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), keySuffix) {
			keyName := file.Name()
			certName := strings.TrimSuffix(file.Name(), keySuffix) + certSuffix
			_, err := os.Stat(certName)
			if err == nil {
				// found a key and a cert that match the mkcert pattern
				return certName, keyName, nil
			}
		}
	}

	return "", "", nil
}
