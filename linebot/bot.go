package linebot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/line/line-bot-sdk-go/linebot"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

var config Config

// Config LINE setting
type Config struct {
	LineChannelSecret string
	LineChannelToken  string
	LineGroupID       string
	FirebaseBaseURL   string
	FirebaseSecret    string
}

type googleHomeMessage struct {
	Text string `json:"text"`
}

type firebaseMessage struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
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
		switch event.Source.Type {
		case linebot.EventSourceTypeGroup:
			if event.Source.GroupID == config.LineGroupID || event.Type == linebot.EventTypeMessage {
				if err := handleGroupMessageEvent(ctx, bot, &event); err != nil {
					return appErrorf(err, http.StatusInternalServerError, "Error on handleGroupMessageEvent")
				}
			}
		case linebot.EventSourceTypeUser:
			if event.Type == linebot.EventTypeMessage {
				// echo
				if _, err := bot.ReplyMessage(event.ReplyToken, event.Message).WithContext(ctx).Do(); err != nil {
					return appErrorf(err, http.StatusInternalServerError, "Error on reply message")
				}
			}
		default:
			log.Debugf(ctx, "Got other event!!")
		}
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func handleGroupMessageEvent(ctx context.Context, bot *linebot.Client, event *linebot.Event) error {
	switch msg := event.Message.(type) {
	case *linebot.TextMessage:
		if err := saveMesagge2Firebase(ctx, msg.Text); err != nil {
			return err
		}
		if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("お伝えします")).WithContext(ctx).Do(); err != nil {
			return err
		}
		log.Debugf(ctx, "Got text!! %s", msg.Text)
	default:
		log.Debugf(ctx, "Got other foramt!!")
	}
	return nil
}

func saveMesagge2Firebase(ctx context.Context, msg string) error {
	firebaseMessage := &firebaseMessage{
		Message:   msg,
		Timestamp: time.Now().Unix(),
	}
	body, err := encodeJSON(firebaseMessage)
	if err != nil {
		return err
	}
	// use firebase REST API
	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/linebot/receive.json?auth=%s", config.FirebaseBaseURL, config.FirebaseSecret),
		body)
	if err != nil {
		return err
	}
	resp, err := createHTTPClient(ctx).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
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
