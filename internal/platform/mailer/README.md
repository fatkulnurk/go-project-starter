# `internal/platform/mailer/`

**Technical infrastructure** — `mailer.MailSender` driver implementations, the
factory that picks one from `MAIL_DRIVER`, MIME rendering, and the shared email
layout.

## Drivers

| Driver  | Type   | Constructor  | Notes                                                        |
|---------|--------|--------------|--------------------------------------------------------------|
| `log`   | `Log`  | `NewLog()`   | logs instead of sending; default dev driver                  |
| `smtp`  | `SMTP` | `NewSMTP(from, fromName, cfg)` | pooled connections, lazy creation, retry on network errors, idle sweep, `Close` required |
| `ses`   | `SES`  | `NewSES(from, fromName, cfg)` | Amazon SESv2 raw MIME; empty credentials fall back to the AWS credential chain |

## Factory

`New(cfg config.MailConfig) (mailer.MailSender, error)` — returns the sender for
`cfg.Driver`; unsupported drivers return an error.

## MIME

`buildMIME(from, fromName, msg)` renders a full RFC-822 message (headers + body)
including attachments as a `multipart/mixed` payload. It is shared by the SMTP
and SES drivers so attachment behavior is identical everywhere. Headers are
sanitized against header injection and non-ASCII values use RFC-2047 encoding.

## SMTP pool

Configured by `MAIL_SMTP_POOL_SIZE`, `MAIL_SMTP_SSL` (`none`, `tls` for
implicit TLS / port 465, or `starttls` default), modeled after `knadh/smtppool`.
A background sweeper closes connections idle longer than 30s; borrows after
`Close` return `ErrSMTPClosed`.

## Email layout

`NewEmailLayout() (*html/template.Template, error)` parses the embedded
`layout.html` shell defining a `layout` template with a `content` block.
Modules attach their own content templates via `ParseFS`.

## Usage

```go
sender, err := mailer.New(cfg.Mail)
if err != nil { return err }
defer sender.Close()

err = sender.Send(ctx, mailer.Message{
    To:      []string{"user@example.com"},
    Subject: "Verify your email",
    HTML:    htmlBody,
})
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
