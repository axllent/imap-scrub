package lib

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

// DeletedAttachment struct
type DeletedAttachment struct {
	Filename string
	MimeType string
	Size     string
}

// HandleMessage will process an imap message
func HandleMessage(msg *imap.Message, rule Rule) (string, int, error) {
	var section imap.BodySectionName

	imap.CharsetReader = charset.Reader

	if msg == nil {
		return "", 0, fmt.Errorf("Server didn't returned message")
	}

	r := msg.GetBody(&section)
	if r == nil {
		return "", 0, fmt.Errorf("Server didn't returned message body")
	}

	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		return "", 0, err
	}

	var b bytes.Buffer

	mw, err := mail.CreateWriter(&b, mr.Header)
	if err != nil {
		return "", 0, err
	}

	defer mw.Close()

	iw, err := mw.CreateInline()
	if err != nil {
		return "", 0, err
	}

	deleted := []DeletedAttachment{}

	froms := msg.Envelope.From

	inlineClosed := false

	// count the number of message parts. If message has none, return error
	msgParts := 0

	emailAddress := "no-email"
	if len(froms) > 0 {
		emailAddress = froms[0].Address()
	}

	// Read each mail's part
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			// unhandled characters etc
			if strings.Contains(err.Error(), "unexpected EOF") {
				return "", 0, fmt.Errorf("No attachments, skipping")
			}
			return "", 0, err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct := p.Header.Get("Content-Type")

			msgParts++

			if strings.HasPrefix(ct, "text") {
				var th mail.InlineHeader

				th.Set("Content-Type", ct)

				if hdr := p.Header.Get("Content-Disposition"); hdr != "" {
					th.Set("Content-Disposition", hdr)
				}
				if hdr := p.Header.Get("Content-Transfer-Encoding"); hdr != "" {
					th.Set("Content-Transfer-Encoding", hdr)
				}

				w, err := iw.CreatePart(th)
				if err != nil {
					// eg: unhandled charset "windows-1252"
					if !strings.Contains(err.Error(), "charset") {
						return "", 0, err
					}

					// bit of a hack - change charset to utf-8 if unsupported
					th.Del("Content-Type")
					if strings.Contains(ct, "text/plain") {
						th.Set("Content-Type", "text/plain; charset=\"UTF-8\"")
					} else {
						th.Set("Content-Type", "text/html; charset=\"UTF-8\"")
					}

					w, err = iw.CreatePart(th)
					if err != nil {
						return "", 0, err
					}
				}

				b, err := io.ReadAll(p.Body)
				if err != nil {
					return "", 0, err
				}
				_, _ = w.Write(b)
				_ = w.Close()
			} else if strings.HasPrefix(ct, "image/") {
				// filename logic taken from https://github.com/emersion/go-message/blob/master/mail/attachment.go
				_, params, err := h.ContentDisposition()
				if err != nil {
					return "", 0, err
				}

				filename, ok := params["filename"]
				if !ok {
					// Using "name" in Content-Type is discouraged
					_, params, err = h.ContentType()
					filename = params["name"]
				}

				b, err := io.ReadAll(p.Body)
				if err != nil {
					return "", 0, err
				}
				if rule.SaveAttachments() {
					if filename, err = SaveAttachment(b, emailAddress, filename, msg.Envelope.Date); err != nil {
						return "", 0, err
					}
				}

				ctString := strings.Fields(ct)[0]
				contentType := ctString[0 : len(ctString)-1]
				size := ByteCountSI(uint32(len(b)))

				deleted = append(deleted, DeletedAttachment{filename, contentType, size})
			}

		case *mail.AttachmentHeader:
			if !inlineClosed {
				_ = iw.Close() // ensures that no further inline html/text can be written
				inlineClosed = true
			}
			filename, err := h.Filename()
			if err != nil {
				return "", 0, err
			}

			if strings.HasSuffix(filename, "-attachments-deleted.txt") {
				continue
			}

			if filename == "" {
				// plain text messages
				filename = "text.txt"
			}

			b, _ := io.ReadAll(p.Body)

			if rule.SaveAttachments() {
				if filename, err = SaveAttachment(b, emailAddress, filename, msg.Envelope.Date); err != nil {
					return "", 0, err
				}
			}

			msgParts++

			ct := p.Header.Get("Content-Type")

			size := ByteCountSI(uint32(len(b)))

			deleted = append(deleted, DeletedAttachment{filename, ct, size})
		}
	}

	if !inlineClosed {
		_ = iw.Close() // ensures that no further inline html/text can be written
	}

	if len(deleted) > 0 {
		if rule.RemoveAttachments() {
			Log.NoticeF(" - Removed %d attachments", len(deleted))
		}

		attachmentText := fmt.Sprintf("Attachments were deleted by imap-scrub on the %s", time.Now().Format("2006-01-02 3:4:5pm"))
		if rule.SaveAttachments() {
			attachmentText += " and moved to the following locations"
		}
		attachmentText += ":\n\n"

		deletedText := ""
		for _, a := range deleted {
			deletedText += fmt.Sprintf(" - %s [%s]\n", a.Filename, a.Size)
		}

		attachmentText += deletedText

		var ah mail.AttachmentHeader

		ah.Set("Content-Type", fmt.Sprintf("text/plain; name=\"%d-attachments-deleted.txt\"", len(deleted)))

		aw, err := mw.CreateAttachment(ah)

		if err != nil {
			return "", 0, err
		}

		if _, err := aw.Write([]byte(attachmentText)); err != nil {
			return "", 0, err
		}
		_ = aw.Close()
		_ = mw.Close()
	}

	if msgParts == 0 {
		// github.com/emersion/go-message/mail - This package assumes that a mail message contains
		// one or more text parts and zero or more attachment parts.
		// we did not find any message parts (inline text only?)
		return "", 0, fmt.Errorf("No attachments")
	}

	return b.String(), len(deleted), nil
}
