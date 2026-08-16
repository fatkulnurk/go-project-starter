# `internal/application/mailer/`

**Cross-cutting capability** — the mail contract. Business modules only know
`MailSender`; the driver (`log`, `smtp`, `ses`) is chosen in the composition
root.

## Key types

| Symbol      | Purpose                                                    |
|-------------|------------------------------------------------------------|
| `MailSender`| delivers one `Message` to all recipients                    |
| `Message`   | a single email: `To`, optional `From`/`FromName`, `Subject`, `Text`/`HTML`, `Attachments` |
| `Attachment`| a file: `Filename`, `Content` (`[]byte`), optional `ContentType` (inferred from the filename when empty) |

`Send(ctx, msg)` returns an error when the driver rejects the message (bad
address, network failure, ...); `Close()` releases pooled resources (SMTP
connections) and is a no-op for stateless drivers.

## Usage

```go
err := sender.Send(ctx, mailer.Message{
    To:      []string{"user@example.com"},
    Subject: "Verify your email",
    HTML:    htmlBody,
    Text:    textBody,
    Attachments: []mailer.Attachment{{
        Filename: "report.pdf",
        Content:  pdfBytes,
    }},
})
```

Implemented by `internal/platform/mailer` (`NewLog`, `NewSMTP`, `NewSES`) behind
the `mailer.New` factory.

## Dependency rules

Vendor-free contract; imports stdlib only.
