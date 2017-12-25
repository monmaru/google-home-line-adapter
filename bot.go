package linebot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/line/line-bot-sdk-go/linebot"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

var (
	myGroupID string
)

type googleHomeMessage struct {
	Text string `json:"text"`
}

type botHandler func(context.Context, *linebot.Client, http.ResponseWriter, *http.Request) *appError

func withBotHandler(h botHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		bot, err := createBotClient(ctx)
		if err != nil {
			log.Criticalf(ctx, "linebot init error: %#v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if ae := h(ctx, bot, w, r); ae != nil {
			msg := fmt.Sprintf("Handler error: status code: %d, message: %s, underlying err: %#v", ae.Code, ae.Message, ae.Error)
			w.WriteHeader(ae.Code)
			log.Errorf(ctx, msg)
		}
	}
}

func init() {
	myGroupID = os.Getenv("LINE_GROUP_ID")
	http.Handle("/", router())
}

func router() *mux.Router {
	router := mux.NewRouter()
	// health check
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "pong")
	}).Methods("GET")
	router.HandleFunc("/googlehome/in", withBotHandler(handleGoogleHomeMessage)).Methods("POST")
	router.HandleFunc("/googlehome/out", withBotHandler(handleLineEvent)).Methods("POST")
	return router
}

func decodeJSON(rc io.ReadCloser, out interface{}) error {
	defer rc.Close()
	return json.NewDecoder(rc).Decode(&out)
}

func createBotClient(c context.Context) (*linebot.Client, error) {
	return linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
		linebot.WithHTTPClient(urlfetch.Client(c)))
}

func handleGoogleHomeMessage(ctx context.Context, bot *linebot.Client, w http.ResponseWriter, r *http.Request) *appError {
	var msg googleHomeMessage
	if err := decodeJSON(r.Body, &msg); err != nil {
		return appErrorf(err, http.StatusInternalServerError, "Error on decode JSON")
	}

	if _, err := bot.PushMessage(myGroupID, linebot.NewTextMessage(msg.Text)).WithContext(ctx).Do(); err != nil {
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
			if event.Source.GroupID == myGroupID || event.Type == linebot.EventTypeMessage {
				if err := handleGroupMessageEvent(ctx, bot, event); err != nil {
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
		if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(msg.Text)).WithContext(ctx).Do(); err != nil {
			return err
		}
		log.Debugf(ctx, "Got text!! %s", msg.Text)
	default:
		log.Debugf(ctx, "Got other foramt!!")
	}
	return nil
}

type appError struct {
	Error   error
	Message string
	Code    int
}

func appErrorf(err error, code int, format string, v ...interface{}) *appError {
	return &appError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    code,
	}
}
