package firebase

import (
	"context"

	firebaseV4 "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

func NewClient(
	config *firebaseV4.Config,
	opt option.ClientOption,
) (*auth.Client, error) {
	app, err := firebaseV4.NewApp(context.Background(), config, opt)

	if err != nil {
		return nil, err
	}

	authClient, err := app.Auth(context.Background())
	if err != nil {
		return nil, err
	}

	return authClient, nil
}
