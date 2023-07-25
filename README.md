# IMAP-Scrub

[![Go Report Card](https://goreportcard.com/badge/github.com/axllent/imap-scrub)](https://goreportcard.com/report/github.com/axllent/imap-scrub)

A command-line utility (Linux, Mac & Windows) to reduce the size of your IMAP mailbox through a series of pre-defined rules. Each rule contain a series of search modifiers, and one or two actions (`delete`, `remove_attachments`, `save_attachments`).

I wrote this tool because I receive many emails with attachments that I need for a limited time only. After a year or two, these attachments do nothing more than take up space, however I did not want to just delete the emails themselves as many contain information that I would rather keep. In another example, certain emails I just do not want to keep at all after a certain period (social media notifications etc).


## Usage options

```
Usage: imap-scrub [options] <config.yml>

Options:
  -y, --yes            do actions (based on config rule actions)
  -m, --mailboxes      list mailboxes on server (helpful for configuration)
  -p, --print-config   print config
  -u, --update         update to latest release version
  -v, --version        show app version
```

## Configuration

Each mail account should have a yaml configuration file. IMAP-Scrub does not currently support OAUTH, so username/password IMAP login is required.

## Example config


```yaml
name: My Gmail Account
host: imap.gmail.com
user: example-user@gmail.com
pass: MySecretPassword123
save_path: /home/me/email-files
rules:
  - mailbox: "[Gmail]/All Mail"  # IMAP mailbox name
    min_size: 5120               # minimum size in kB
    older_than: 365              # days
    actions: remove_attachments
  - mailbox: "[Gmail]/All Mail"
    from: invitations@linkedin.com
    older_than: 30
    actions: delete
  - mailbox: "[Gmail]/All Mail"
    from: myclient@example.com
    min_size: 512 
    older_than: 90
    actions: save_attachments, remove_attachments
```

See [All yaml config options](#all-yaml-config-options) below for more info.


## Installing

Download the [latest binary release](https://github.com/axllent/imap-scrub/releases/latest) for your system, 
or build from source `go install github.com/axllent/imap-scrub@latest`(go >= 1.11 required)


## All yaml config options

```yaml
name:      string # reference name of this account
host:      string # IMAP hostname
ssl:       true   # use SSL (default true)
port:      993    # IMAP port number (default 993 if SSL is true, else 143)
user:      string # IMAP username
pass:      string # IMAP password
save_path: string # local directory to save attachments (default current dir)
use_trash: false  # see below
rules:
  - mailbox:         string # IMAP mailbox name see below)
    min_size:        0      # minimum message size in kB
    older_than:      0      # older than x days
    from:            string # match "From" field
    to:              string # match "To" field
    subject:         string # match email subject
    body:            string # match email body
    text:            string # match email message
    actions:         string # see below
    include_unread:  false  # include unread messages (default false)
    include_starred: false  # include starred messages (default false)
```


### Option: `mailbox`

The mailbox you wish to search. On standard IMAP servers this is probably `INBOX`. 

On Gmail this is possibly `[Gmail]/All Mail` or `[Google Mail]/All Mail`, but may differ based on your selected language. To list the mailboxes on your IMAP server to make a choice, run `imap-scrub -m <your-config.yml>` which will print out all mailboxes in your account.


### Option: `use_trash`

If `use_trash` is set to `true`, and your IMAP returns a trash mailbox, then deleted messages will be moved into this mailbox. **Note** that Gmail does not support IMAP delete, so `use_trash` will always be set to `true` for Gmail.


### Option: `actions`

There are three possible actions, namely:

- `save_attachments` will save any attachments to `save_path`
- `remove_attachments` will remove the all attachments and inline images from the original email 
- `delete` will simply delete the email

The `actions:` config may include a combination of `save_attachments` and one other (comma-separated), eg :`actions: save_attachments, remove_attachments`. 

**Note** that you cannot combine `remove_attachments` and `delete`.
