package linebot

import (
	"context"
	"fmt"
	"net/http"

	"github.com/line/line-bot-sdk-go/linebot"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

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
			w.WriteHeader(ae.Code)
			msg := fmt.Sprintf("Handler error: status code: %d, message: %s, underlying err: %#v", ae.Code, ae.Message, ae.Error)
			log.Errorf(ctx, msg)
		}
	}
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
