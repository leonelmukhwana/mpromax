package notifications

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMProvider struct {
	client *messaging.Client
}

func NewFCMProvider(credentialsFile string) (*FCMProvider, error) {
	ctx := context.Background()

	// If the file path is empty, we return a clear error
	if credentialsFile == "" {
		return nil, fmt.Errorf("firebase credentials file path is empty")
	}

	opt := option.WithCredentialsFile(credentialsFile)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %v", err)
	}

	return &FCMProvider{client: client}, nil
}

func (p *FCMProvider) SendPush(ctx context.Context, token, title, body string, data map[string]string) error {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  data,
		Token: token,
	}

	_, err := p.client.Send(ctx, message)
	return err
}
