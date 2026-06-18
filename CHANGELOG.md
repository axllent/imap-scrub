# Changelog

## [0.0.7]

- Fix `older_than` to use the IMAP internal delivery date instead of the sender-controlled `Date` header
- Normalize the cutoff to local midnight and re-verify each fetched message before delete or attachment actions (IMAP `BEFORE` is date-only and can return borderline matches)
- Clarify debug output when messages are skipped: show internal date and cutoff in local time, and note when the envelope `Date` header differs


## [0.0.6]

- Avoid altering messages that don't have attachments (#7) thanks to @michalfapso


## [0.0.5]

- Write full output paths of saved attachments into "%d-attachments-deleted.txt"
- Bump minimum Go version, update Go modules
- Replace deprecated ioutil references


## [0.0.4]

- Fix confusing include_unread output (#5)
- Code cleanup
- Add GitHub actions workflow
- New internal self-updater
- Update Go modules


## [0.0.3]

- Upgrade modules / dependencies


## [0.0.2]

- Support darwin arm64
- Remove ineffectual assignment to inlineClosed
- Handle trash detection errors


## [0.0.1]

- Initial release
