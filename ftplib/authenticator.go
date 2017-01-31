package ftplib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Authenticator contains credentials.
// Implements https://godoc.org/github.com/goftp/server#Auth
type Authenticator struct {
	credentials map[string]string
}

// AuthenticatorFromFile returns an Authenticator with credentials parsed from the given file path.
// The file must contain one credential entry per line with username and password separated by a `:`.
func AuthenticatorFromFile(path string) (Authenticator, error) {
	auth := Authenticator{make(map[string]string)}

	file, err := os.Open(path)
	if err != nil {
		return auth, err
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				auth.credentials[parts[0]] = parts[1]
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return auth, err
		}
	}
	if len(auth.credentials) == 0 {
		return auth, fmt.Errorf("No credentials found in: %s", path)
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
