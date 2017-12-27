package gaebot

import (
	"os"

	"github.com/monmaru/google-home-line-adapter/linebot"
)

func init() {
	linebot.Run(loadEnv())
}

func loadEnv() linebot.Config {
	return linebot.Config{
		LineChannelSecret: os.Getenv("LINE_CHANNEL_SECRET"),
		LineChannelToken:  os.Getenv("LINE_CHANNEL_TOKEN"),
		LineGroupID:       os.Getenv("LINE_GROUP_ID"),
		FirebaseBaseURL:   os.Getenv("FIREBASE_BASE_URL"),
		FirebaseSecret:    os.Getenv("FIREBASE_SECRET"),
	}
}
