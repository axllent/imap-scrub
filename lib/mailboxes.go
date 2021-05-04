package lib

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// ListMailboxes returns a list of Mailboxs on the server
func ListMailboxes(cReader *client.Client) {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- cReader.List("", "*", mailboxes)
	}()

	Log.InfoF("Mailboxes on %s\n", Config.Name)
	for m := range mailboxes {
		if !InStringSlice("\\Noselect", m.Attributes) {
			Log.Info(" - " + m.Name)
		}
	}

	if err := <-done; err != nil {
		Log.ErrorF("%v\n", err)
	}
}

// DetectTrash will return the trash folder of a Gmail account, if appliccable
// Gmail only supports moving to the trash
func DetectTrash(cReader *client.Client) (string, error) {
	if !Config.UseTrash && Config.Host != "imap.gmail.com" {
		return "", nil
	}

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- cReader.List("", "*", mailboxes)
	}()

	var trashMailbox = ""
	for m := range mailboxes {
		if InStringSlice("\\Trash", m.Attributes) {
			Log.DebugF("Deleted messages will be moved to \"%s\"", m.Name)
			trashMailbox = m.Name
		}
	}

	if err := <-done; err != nil {
		Log.ErrorF("%v\n", err)
	}

	if trashMailbox != "" {
		return trashMailbox, nil
	}

	return "", nil
}
