# `internal/application/sms/`

**Cross-cutting capability** — the SMS contract. Business modules only know
`Sender`; the driver (`log`, `twilio`) is chosen in the composition root.

## Key types

| Symbol    | Purpose                                                    |
|-----------|------------------------------------------------------------|
| `Sender`  | delivers one SMS message                                   |
| `Message` | a single SMS: `To`, optional `From` (sender id), `Body`    |

`Send(ctx, msg)` returns an error when the backend rejects the send (e.g.
invalid credentials, network failure).

## Usage

```go
err := sender.Send(ctx, sms.Message{
    To:   "+6281234567890",
    Body: "Your code is 123456",
})
```

Implemented by `internal/platform/sms` (`NewLog`, `NewTwilio`) behind the
`sms.New` factory.

## Dependency rules

Vendor-free contract; imports stdlib only.
