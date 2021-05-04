package lib

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	mboxlib "github.com/emersion/go-mbox"
)

var resultCount = 1

// PrettyPrint outputs a JSON-encoded representation of an interface
func PrettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

// CreateDir will check if a directory exists, and create it if not
func CreateDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// PrintHdrDetails returns a IMAP search result
func PrintHdrDetails(msg *imap.Message) {
	e := msg.Envelope
	from := TruncateFromAddress(e.From)
	hrSize := ByteCountSI(msg.Size)
	starred := " "
	if InStringSlice("\\Flagged", msg.Flags) {
		starred = "*"
	}

	Log.InfoF("#%-4d %s  %s %-62s %s%7s", resultCount, e.Date.Format("02-Jan-06"), from, Truncate(e.Subject, 60), starred, hrSize)
	resultCount++
}

// Truncate will return a truncates string
func Truncate(raw string, length int) string {
	var numRunes = 0
	for index := range raw {
		numRunes++
		if numRunes > length {
			return raw[:index-3] + "..."
		}
	}
	return raw
}

// TruncateFromAddress returns a formatted and truncated From address
func TruncateFromAddress(from []*imap.Address) string {
	if len(from) == 0 {
		return "Unknown sender"
	}
	email := fmt.Sprintf(" <%s>", from[0].Address())

	emailLength := len(email)

	remaining := 45 - emailLength

	name := ""
	if remaining > 5 {
		name = Truncate(from[0].PersonalName, remaining)
	}

	return fmt.Sprintf("%-47s", strings.TrimSpace(name+email))
}

// ByteCountSI returns a human-readable size from bytes
func ByteCountSI(b uint32) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

// InStringSlice returns whether a value is in a string slice
func InStringSlice(val string, slice []string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// SaveAttachment will save an attachment to <outdir>/<email>/<hash>-<filename>
func SaveAttachment(b []byte, emailAddress, fileName string, timestamp time.Time) error {
	fileName = filepath.Base(fileName)

	if fileName == "" {
		return fmt.Errorf("Filename empty, not saving")
	}

	h := sha1.New()
	h.Write(b)
	hashb := h.Sum(nil)

	hashed := fmt.Sprintf("%x-%s", hashb[0:3], fileName)

	outDir := path.Join(Config.SavePath, emailAddress)
	if err := CreateDir(outDir); err != nil {
		return err
	}

	outFile := path.Join(outDir, hashed)
	if FileExists(outFile) {
		Log.WarningF(" - %s/%s already exists", emailAddress, hashed)
		return nil
	}
	file, err := os.OpenFile(
		outFile,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0664,
	)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write bytes to file
	bytesWritten, err := file.Write(b)
	if err != nil {
		return err
	}
	// write a copy of the attachment
	bytes := uint32(bytesWritten)

	// set timestamp
	_ = os.Chtimes(outFile, timestamp, timestamp)

	Log.NoticeF(" - Saved %s/%s (%s)", emailAddress, hashed, ByteCountSI(bytes))

	return nil
}

func CreateMBOX(mailboxName string) (*mboxlib.Writer, error) {
	//func CreateMBOX(mailboxName string) (*os.File, error) {
	var mailboxParts = strings.Split(mailboxName, "/")

	outDir := path.Join(Config.SavePath, path.Join(mailboxParts...))
	if err := CreateDir(outDir); err != nil {
		return nil, err
	}

	outFile := path.Join(outDir, "mbox")
	if FileExists(outFile) {
		Log.WarningF(" - File '%s/%s' already exists", outDir, "mbox")
		return nil, fmt.Errorf("Filename exists")
	}

	file, err := os.OpenFile(
		outFile,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0664,
	)
	if err != nil {
		return nil, err
	}
	//defer file.Close()

	mboxWriter := mboxlib.NewWriter(file)

	return mboxWriter, nil
	//return file, nil
}
