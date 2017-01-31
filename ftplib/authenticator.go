package ftplib

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// Authenticator contains credentials.
// Implements https://godoc.org/github.com/goftp/server#Auth
type Authenticator struct {
	credentials map[string]string
}

// AuthenticatorFromFile returns an Authenticator with credentials parsed from the given file path.
// The file must contain one credential pair per line where username and password is separated by a `:`.
func AuthenticatorFromFile(path string) (Authenticator, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return Authenticator{}, err
	}
	return AuthenticatorFromString(string(raw))
}

// AuthenticatorFromString returns an Authenticator whose credentials where parsed from the given string.
// The contents must contain one credential pair per line where username and password is separated by a `:`.
func AuthenticatorFromString(contents string) (Authenticator, error) {
	auth := Authenticator{make(map[string]string)}

	lines := strings.Split(contents, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				auth.credentials[parts[0]] = parts[1]
			}
		}
	}
	if len(auth.credentials) == 0 {
		return auth, fmt.Errorf("No credentials found")
	}
	return auth, nil
}

func (c Authenticator) CheckPasswd(username, password string) (bool, error) {
	for user, pass := range c.credentials {
		if username == user && password == pass {
			return true, nil
		}
	}
	return false, fmt.Errorf("Unknown credentials: %q:%q", username, password)
}
