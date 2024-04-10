// Package main is the main application
package main

import (
	"bytes"
	"fmt"
	"net/textproto"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/axllent/imap-scrub/lib"
	"github.com/axllent/imap-scrub/lib/updater"
	"github.com/emersion/go-imap"
	move "github.com/emersion/go-imap-move"
	"github.com/emersion/go-imap/client"
	"github.com/spf13/pflag"
)

var (
	cReader       *client.Client
	cWriter       *client.Client
	doActions     bool
	deleteActions bool
	appVersion    = "dev"
)

func main() {
	var configFile string
	var listMailboxes, printConfig, showVersion, update bool
	var headersOnly = true

	flag := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	// set the default help
	flag.Usage = func() {
		fmt.Printf("IMAP Scrub - https://github.com/axllent/imap-scrub\n\n")
		fmt.Printf("Usage: %s [options] <config.yml>\n", os.Args[0])
		fmt.Println("\nOptions:")
		flag.SortFlags = false
		flag.PrintDefaults()
	}

	// add options
	flag.BoolVarP(&doActions, "yes", "y", false, "do actions (based on config rule actions)")
	flag.BoolVarP(&listMailboxes, "mailboxes", "m", false, "list mailboxes on server (helpful for configuration)")
	flag.BoolVarP(&printConfig, "print-config", "p", false, "print config")
	flag.BoolVarP(&update, "update", "u", false, "update to latest release version")
	flag.BoolVarP(&showVersion, "version", "v", false, "show app version")
	// avoid 'pflag: help requested' error, as help will be defined later by cobra cmd.Execute()
	flag.BoolP("help", "h", false, "")

	_ = flag.Parse(os.Args[1:])

	// parse arguments
	args := flag.Args()

	if showVersion {
		fmt.Printf("%s %s compiled with %s on %s/%s\n",
			os.Args[0], appVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)

		latest, _, _, err := updater.GithubLatest("axllent/imap-scrub", "imap-scrub")
		if err == nil && updater.GreaterThan(latest, appVersion) {
			fmt.Printf(
				"\nUpdate available: %s\nRun `%s version -u` to update (requires read/write access to install directory).\n",
				latest,
				os.Args[0],
			)
		}
		os.Exit(0)
	}

	if update {
		rel, err := updater.GithubUpdate("axllent/imap-scrub", "imap-scrub", appVersion)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		fmt.Printf("Updated %s to version %s\n", os.Args[0], rel)
		os.Exit(0)
	}

	if len(args) != 1 {
		flag.Usage()
		os.Exit(0)
	}

	configFile = args[0]

	lib.ReadConfig(configFile)

	if printConfig {
		lib.Config.Pass = "**********"
		lib.PrettyPrint(lib.Config)
		os.Exit(0)
	}

	imapServer := fmt.Sprintf("%s:%d", lib.Config.Host, *lib.Config.Port)

	lib.Log.DebugF("Connecting to %s...", imapServer)

	cReader = lib.Connect()
	cWriter = lib.Connect()

	// Don't forget to logout afterwards
	defer cReader.Logout()
	defer cWriter.Logout()

	// Login
	if err := cReader.Login(lib.Config.User, lib.Config.Pass); err != nil {
		lib.Log.Errorf("%v", err)
		os.Exit(2)
	}
	if err := cWriter.Login(lib.Config.User, lib.Config.Pass); err != nil {
		lib.Log.Errorf("%v", err)
		os.Exit(2)
	}

	if listMailboxes {
		lib.ListMailboxes(cReader)
		os.Exit(0)
	}

	trashMailbox, err := lib.DetectTrash(cReader)
	if err != nil {
		lib.Log.Error(err.Error())
		os.Exit(2)
	}

	for _, rule := range lib.Config.Rules {
		// If we are removing or saving attachments, then pull the whole message in the search
		if doActions && (rule.RemoveAttachments() || rule.SaveAttachments()) {
			headersOnly = false
		}

		sFilters := []string{}

		// Select mailbox
		mbox, err := cReader.Select(rule.Mailbox, true)
		if err != nil {
			lib.Log.Errorf(err.Error())
			continue
		}

		// Get the last message
		if mbox.Messages == 0 {
			lib.Log.DebugF("No messages matching search in %s", rule.Mailbox)
			continue
		}

		seqSet := new(imap.SeqSet)
		now := time.Now()

		// search criteria
		crit := imap.SearchCriteria{}

		if !rule.IncludeUnread {
			// only seen messages
			sFilters = append(sFilters, "read")
			crit.WithFlags = []string{"\\Seen"}
		}

		if !rule.IncludeStarred {
			// skip starred
			sFilters = append(sFilters, "unstarred")
			crit.WithoutFlags = []string{"\\Flagged"}
		}

		if rule.OlderThan > 0 {
			sFilters = append(sFilters, fmt.Sprintf("older: %d days", rule.OlderThan))
			crit.SentBefore = now.Add(-(time.Duration(rule.OlderThan) * 24 * time.Hour))
		}

		if rule.Size > 0 {
			sFilters = append(sFilters, fmt.Sprintf("larger: %s", lib.ByteCountSI(rule.Size)))
			crit.Larger = rule.Size
		}

		if rule.Text != "" {
			sFilters = append(sFilters, fmt.Sprintf("containing: \"%s\"", rule.Text))
			crit.Text = append(crit.Text, rule.Text)
		}
		if rule.Body != "" {
			sFilters = append(sFilters, fmt.Sprintf("body: \"%s\"", rule.Body))
			crit.Body = append(crit.Body, rule.Body)
		}

		headerSearch := textproto.MIMEHeader{}

		if rule.From != "" {
			sFilters = append(sFilters, fmt.Sprintf("from: \"%s\"", rule.From))
			headerSearch["From"] = append(headerSearch["From"], rule.From)
		}

		if rule.To != "" {
			sFilters = append(sFilters, fmt.Sprintf("to: \"%s\"", rule.To))
			headerSearch["To"] = append(headerSearch["To"], rule.To)
		}
		if rule.Subject != "" {
			sFilters = append(sFilters, fmt.Sprintf("subject: \"%s\"", rule.Subject))
			headerSearch["Subject"] = append(headerSearch["Subject"], rule.Subject)
		}

		if len(headerSearch) > 0 {
			crit.Header = headerSearch
		}

		lib.Log.DebugF("Searching \"%s\" for %s", rule.Mailbox, strings.Join(sFilters, ", "))

		// search
		searchRes, err := cReader.UidSearch(&crit)
		if err != nil {
			lib.Log.Errorf(err.Error())
			continue
		}

		if len(searchRes) <= 0 {
			lib.Log.DebugF("%s returned 0 results from the last %d days", rule.Mailbox, rule.OlderThan)
			continue
		}

		_, err = cWriter.Select(rule.Mailbox, false)
		if err != nil {
			lib.Log.Errorf(err.Error())
			continue
		}

		// add messages to the queue
		for _, sr := range searchRes {
			seqSet.AddNum(sr)
		}

		// Get the whole message body
		var section imap.BodySectionName

		if headersOnly {
			// list-only don't need to download entire mail, just headers
			section.Specifier = imap.HeaderSpecifier
		}

		items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, imap.FetchRFC822Size, section.FetchItem()}

		messages := make(chan *imap.Message, 1)

		go func() {
			if err := cReader.UidFetch(seqSet, items, messages); err != nil {
				lib.Log.Errorf(err.Error())
				os.Exit(2)
			}
		}()

		// total size of all matching emails
		var totalSize uint32

		for msg := range messages {
			// print search result
			lib.PrintHdrDetails(msg)

			totalSize = totalSize + msg.Size

			deletedAttachments := 0

			if doActions && (rule.RemoveAttachments() || rule.SaveAttachments()) {
				raw, attachments, err := lib.HandleMessage(msg, rule)
				if err != nil {
					lib.Log.Errorf("%s", err)
					continue
				}

				if attachments == 0 {
					lib.Log.Warningf("no attachments detected")
					continue
				}

				// cReader.SetDebug(os.Stdout)
				literal := bytes.NewBufferString(raw)

				if attachments > 0 && rule.RemoveAttachments() {
					// create a new message and copy envelope & flags
					if err := cWriter.Append(rule.Mailbox, msg.Flags, msg.Envelope.Date, literal); err != nil {
						lib.Log.Errorf(err.Error())
						continue
					}
				}

				deletedAttachments = attachments
			}

			if doActions && (rule.RemoveAttachments() && deletedAttachments > 0 || rule.Delete()) {
				seqSet := new(imap.SeqSet)
				seqSet.AddNum(msg.Uid)

				if trashMailbox != "" {
					// move to Bin
					mover := move.NewClient(cWriter)
					if err := mover.UidMove(seqSet, trashMailbox); err != nil {
						lib.Log.Errorf(err.Error())
						continue
					}
					lib.Log.NoticeF(" - Moved original message to trash")
				} else {
					// delete original
					item := imap.FormatFlagsOp(imap.AddFlags, true)
					flags := []interface{}{imap.DeletedFlag}
					if err := cWriter.UidStore(seqSet, item, flags, nil); err != nil {
						lib.Log.Errorf(err.Error())
						continue
					}
					if err := cWriter.Expunge(nil); err != nil {
						lib.Log.Errorf(err.Error())
						continue
					}
					lib.Log.NoticeF(" - Deleted original message")
				}
			}
		}

		if totalSize > 0 {
			lib.Log.DebugF("=====\nTotal size: %s\n=====\n", lib.ByteCountSI(totalSize))
		}
	}
}
