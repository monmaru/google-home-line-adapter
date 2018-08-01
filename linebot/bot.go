package linebot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/line/line-bot-sdk-go/linebot"
	"golang.org/x/oauth2/google"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"gopkg.in/zabawaba99/firego.v1"
)

var config Config

// Config LINE setting
type Config struct {
	LineChannelSecret string
	LineChannelToken  string
	LineGroupID       string
	LineBeaconHWID    string
	FirebaseBaseURL   string
}

type googleHomeMessage struct {
	Text string `json:"text"`
}

// Run line bot service
func Run(c Config) {
	config = c
	http.Handle("/", route())
}

func route() *mux.Router {
	router := mux.NewRouter()
	// health check
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "pong")
	}).Methods("GET")
	router.HandleFunc("/googlehome/in", withBotHandler(handleGoogleHomeMessage)).Methods("POST")
	router.HandleFunc("/googlehome/out", withBotHandler(handleLineEvent)).Methods("POST")
	return router
}

func handleGoogleHomeMessage(ctx context.Context, bot *linebot.Client, w http.ResponseWriter, r *http.Request) *appError {
	var msg googleHomeMessage
	if err := decodeJSON(r.Body, &msg); err != nil {
		return appErrorf(err, http.StatusInternalServerError, "Error on decode JSON")
	}

	if _, err := bot.PushMessage(config.LineGroupID, linebot.NewTextMessage(msg.Text)).WithContext(ctx).Do(); err != nil {
		return appErrorf(err, http.StatusInternalServerError, "Error on push message")
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func handleLineEvent(ctx context.Context, bot *linebot.Client, w http.ResponseWriter, r *http.Request) *appError {
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			return appErrorf(err, http.StatusBadRequest, "Invalid signature")
		}
		return appErrorf(err, http.StatusInternalServerError, "Error on parse request")
	}

	for _, event := range events {
		switch event.Type {
		/*
			case linebot.EventTypeBeacon:
				if err := handleBeaconEvent(ctx, bot, &event); err != nil {
					return appErrorf(err, http.StatusInternalServerError, "Error on handleBeaconEvent")
				}
		*/
		case linebot.EventTypeMessage:
			switch event.Source.Type {
			case linebot.EventSourceTypeGroup:
				if event.Source.GroupID == config.LineGroupID {
					if err := handleGroupMessageEvent(ctx, bot, event); err != nil {
						return appErrorf(err, http.StatusInternalServerError, "Error on handleGroupMessageEvent")
					}
				}
			case linebot.EventSourceTypeUser:
				// echo
				if _, err := bot.ReplyMessage(event.ReplyToken, event.Message).WithContext(ctx).Do(); err != nil {
					return appErrorf(err, http.StatusInternalServerError, "Error on reply message")
				}
			default:
				log.Debugf(ctx, "Got other EventSourceType")
			}
		default:
			log.Debugf(ctx, "Got other event!!")
		}
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func handleBeaconEvent(ctx context.Context, bot *linebot.Client, event *linebot.Event) error {
	b := event.Beacon
	if b.Hwid == config.LineBeaconHWID && b.Type == linebot.BeaconEventTypeEnter {
		if err := saveMesagge2Firebase(ctx, "ただいま帰りました"); err != nil {
			return err
		}
	}
	return nil
}

func handleGroupMessageEvent(ctx context.Context, bot *linebot.Client, event *linebot.Event) error {
	switch msg := event.Message.(type) {
	case *linebot.TextMessage:
		log.Debugf(ctx, "Got text!! %s", msg.Text)
		return saveMesagge2Firebase(ctx, msg.Text)
		/*
			if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("お伝えします")).WithContext(ctx).Do(); err != nil {
				return err
			}
		*/
	default:
		log.Debugf(ctx, "Got other foramt!!")
	}
	return nil
}

func saveMesagge2Firebase(ctx context.Context, msg string) error {
	jsonKey, err := ioutil.ReadFile("firebase_service_account.json")
	if err != nil {
		return err
	}

	conf, err := google.JWTConfigFromJSON(
		jsonKey,
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/firebase.database")

	if err != nil {
		return err
	}

	fb := firego.New(
		fmt.Sprintf("%s/linebot/receive", config.FirebaseBaseURL),
		conf.Client(ctx))

	v := map[string]string{
		"message":   msg,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}

	return fb.Set(v)
}

func decodeJSON(rc io.ReadCloser, out interface{}) error {
	defer rc.Close()
	return json.NewDecoder(rc).Decode(&out)
}

func encodeJSON(in interface{}) (io.Reader, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(b), nil
}

func createBotClient(ctx context.Context) (*linebot.Client, error) {
	return linebot.New(
		config.LineChannelSecret,
		config.LineChannelToken,
		linebot.WithHTTPClient(createHTTPClient(ctx)))
}

func createHTTPClient(ctx context.Context) *http.Client {
	return urlfetch.Client(ctx)
}
