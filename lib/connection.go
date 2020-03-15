package lib

import (
	"fmt"
	"os"

	"github.com/emersion/go-imap/client"
)

// Connect returns a *client.Client
func Connect() *client.Client {
	imapServer := fmt.Sprintf("%s:%d", Config.Host, *Config.Port)
	var c *client.Client
	var err error
	if *Config.SSL {
		c, err = client.DialTLS(imapServer, nil)
		if err != nil {
			Log.ErrorF("%v", err)
			os.Exit(2)
		}
	} else {
		c, err = client.Dial(imapServer)
		if err != nil {
			Log.ErrorF("%v", err)
			os.Exit(2)
		}
	}

	return c
}
