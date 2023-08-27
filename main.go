package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

const (
	host = "0.0.0.0"
	port = 6699
)

func middlewareWithLogger(tgBot *BotApi) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			ct := time.Now()
			hpk := s.PublicKey() != nil
			pty, _, _ := s.Pty()
			log.Info("New Connection", "user", s.User(), "remote_addr", s.RemoteAddr().String(), "public_key", hpk, "command", s.Command(), "term", pty.Term, "width", pty.Window.Width, "height", pty.Window.Height)
			tgBot.SendTelegramMessage(fmt.Sprintf("New Connection\nuser: %s\nremote_addr: %s\npublic_key: %t\ncommand: %s\nterm: %s\nwidth: %d\nheight: %d", s.User(), s.RemoteAddr().String(), hpk, s.Command(), pty.Term, pty.Window.Width, pty.Window.Height))
			sh(s)
			log.Info("Connection closed", "remote_addr", s.RemoteAddr().String(), "duration", time.Since(ct))
			tgBot.SendTelegramMessage(fmt.Sprintf("Connection closed\nremote_addr: %s\nduration: %s", s.RemoteAddr().String(), time.Since(ct)))
		}
	}
}

func main() {
	tgBot := InitializeTelegramBot()
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			myCustomBubbleTeaMiddleware(),
			middlewareWithLogger(tgBot),
			//ratelimiter.Middleware(ratelimiter.NewRateLimiter(10, 4, 10)),
		),
	)
	if err != nil {
		log.Error("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}
