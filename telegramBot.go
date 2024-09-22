package main

import (
	"github.com/charmbracelet/log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
)

var chatId, _ = strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)
var token = os.Getenv("TELEGRAM_TOKEN")

func InitializeTelegramBot() *BotApi {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal("telegram bot api error", "error", err)
	}

	bot.Debug = true

	log.Info("Authorized on account", "username", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	botApi := &BotApi{bot: bot}
	botApi.SendTelegramMessage("Bot has been started")

	return botApi
}

type BotApi struct {
	bot *tgbotapi.BotAPI
}

func (botApi *BotApi) SendTelegramMessage(message string) {
	msg := tgbotapi.NewMessage(chatId, message)
	m, _ := botApi.bot.Send(msg)
	log.Info("Message sent", "message", m)
	firstname := m.Chat.FirstName
	if firstname != "N" {
		log.Error("Recipient is not me", "firstname", firstname)
		os.Exit(1)
	}
}
