package mailer

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// SES sends email via Amazon SES (SESv2 API). It supports text, HTML and
// attachments through a raw MIME message.
type SES struct {
	client   *sesv2.Client
	from     string
	fromName string
}

// NewSES builds an SES mail sender.
// It returns an error when cfg.Region is empty or the AWS config cannot be
// loaded; empty credentials fall back to the default AWS credential chain.
func NewSES(from, fromName string, cfg config.SESConfig) (*SES, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("SES_REGION is required")
	}
	var opts []func(*awsconfig.LoadOptions) error
	if cfg.AccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := sesv2.NewFromConfig(awsCfg, func(o *sesv2.Options) {
		o.Region = cfg.Region
	})
	return &SES{client: client, from: from, fromName: fromName}, nil
}

// Send implements mailer.MailSender. Per-message From/FromName override the
// driver defaults when set.
func (s *SES) Send(ctx context.Context, msg mailer.Message) error {
	from, fromName := s.from, s.fromName
	if msg.From != "" {
		from = msg.From
	}
	if msg.FromName != "" {
		fromName = msg.FromName
	}
	data, err := buildMIME(from, fromName, msg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, defaultSendTimeout)
	defer cancel()
	_, err = s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination: &types.Destination{
			ToAddresses: msg.To,
		},
		Content: &types.EmailContent{
			Raw: &types.RawMessage{Data: data},
		},
	})
	if err != nil {
		return fmt.Errorf("ses send to %s: %w", strings.Join(msg.To, ","), err)
	}
	return nil
}

// Close implements mailer.MailSender. The SES client has no pooled resources,
// so nothing is released and nil is always returned.
func (s *SES) Close() error { return nil }
