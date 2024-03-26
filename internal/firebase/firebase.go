// Package firebase handles uploads to firebase storage
package firebase

import (
	"context"
	"fmt"
	"io"
	"os"

	firebase "firebase.google.com/go"

	"github.com/hyperjumptech/bookkeeping/internal/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

var (
	fireLog = log.WithField("module", "firebase")
)

// createApp creates a firebase app instance
func createApp(ctx context.Context) (*firebase.App, error) {
	logf := fireLog.WithField("fn", "createApp")
	var opt option.ClientOption
	var cfg firebase.Config

	cfg.StorageBucket = config.Get("firebase.storage.bucket")

	if config.Get("app.env") == "development" {
		opt = option.WithCredentialsFile("./serviceAccountKey.json")
	} else {
		configJSON := config.Get("firebase.ServiceAccountKey")
		opt = option.WithCredentialsJSON([]byte(configJSON))
	}

	app, err := firebase.NewApp(ctx, &cfg, opt)
	if err != nil {
		logf.Error("error creating app got: ", err)
		return nil, fmt.Errorf("error creating app: %v", err)
	}

	return app, nil
}

// Upload backs up a file to firebase
func Upload(ctx context.Context, fname string) error {
	logf := fireLog.WithField("fn", "uploadFirebase")
	logf.Info("starting upload..")

	app, err := createApp(ctx)
	if err != nil {
		logf.Error("error creating app, got: ", err)
		return fmt.Errorf("error creating app: %v", err)
	}

	client, err := app.Storage(ctx)
	if err != nil {
		logf.Error("error creating storage client, got: ", err)
		return fmt.Errorf("error creating client: %v", err)
	}

	bucket, err := client.DefaultBucket()
	if err != nil {
		logf.Error("error creating client, got: ", err)
		return fmt.Errorf("error creating client: %v", err)
	}

	f, err := os.Open(fname)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer f.Close()

	wc := bucket.Object(fname).NewWriter(ctx)
	count, err := io.Copy(wc, f)
	if err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	log.Info("upload success, size uploaded: ", count)

	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	return nil
}
