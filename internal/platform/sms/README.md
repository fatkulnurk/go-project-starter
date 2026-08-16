# `internal/platform/sms/`

**Technical infrastructure** — `sms.Sender` driver implementations and the
factory that picks one from `SMS_DRIVER`.

## Drivers

| Driver  | Type     | Constructor    | Notes                                                       |
|---------|----------|----------------|-------------------------------------------------------------|
| `log`   | `Log`    | `NewLog()`     | logs instead of sending; default dev driver, needs no credentials |
| `twilio`| `Twilio` | `NewTwilio(from, cfg)` | Twilio Messages API with a 30s client timeout           |

## Factory

`New(cfg config.SMSConfig) (sms.Sender, error)` — returns the sender for
`cfg.Driver`; unsupported drivers return an error.

## Behavior

`Twilio.Send` prefers the messaging service (`TWILIO_MESSAGE_SID`) when
configured, else the per-message `From`, else the configured default `From`;
errors are wrapped with the recipient.

## Usage

```go
sender, err := sms.New(cfg.SMS)
if err != nil { return err }

err = sender.Send(ctx, sms.Message{To: "+6281234567890", Body: "Your code is 123456"})
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
