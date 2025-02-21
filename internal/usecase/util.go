package usecase

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	ulid "github.com/oklog/ulid/v2"
)

var (
	entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}

func convertPNG2JPG(imageBytes []byte) ([]byte, error) {
	contentType := http.DetectContentType(imageBytes)

	switch contentType {
	case "image/png":
		img, err := png.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return nil, err
		}

		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, img, nil); err != nil {
			return nil, err
		}

		return buf.Bytes(), nil
	}

	return nil, fmt.Errorf("unable to convert %#v to jpeg", contentType)
}

func uploadDeckImage(deckCode string) error {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	baseEndpoint := "https://s3.isk01.sakurastorage.jp"
	s3client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = &baseEndpoint
	})

	url := fmt.Sprintf("https://www.pokemon-card.com/deck/deckView.php/deckID/%s.png", deckCode)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	srcImg, _, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}

	var w bytes.Buffer
	err = png.Encode(&w, srcImg)
	if err != nil {
		return err
	}

	imageBytes, err := convertPNG2JPG(w.Bytes())
	if err != nil {
		return err
	}

	params := &s3.PutObjectInput{
		ACL:    "public-read",
		Bucket: aws.String("vsrecorder"),
		Key:    aws.String(fmt.Sprintf("images/decks/%s.jpg", deckCode)),
		Body:   bytes.NewReader(imageBytes),
	}

	_, err = s3client.PutObject(ctx, params)
	if err != nil {
		return err
	}

	return nil
}
