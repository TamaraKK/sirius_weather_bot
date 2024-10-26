package main

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func initTelegramBot(botToken string) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Error creating bot:", err)
	}
	bot.Debug = true
	return bot
}

func handleUpdates(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		log.Printf("Получено обновление: %+v", update)

		if update.Message != nil && update.Message.IsCommand() && update.Message.Command() == "start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет, это бот прогноза погоды. \nВведите название города:")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println("Error sending message:", err)
			}
			continue
		}

		if update.Message != nil && update.Message.Text != "" && update.Message.Command() == "" {
			city := strings.TrimSpace(update.Message.Text)
			valid, err := isValidCity(city)
			if err != nil {
				log.Println("Error validating city:", err)
			}
			if !valid {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный город, попробуйте снова:")
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("Error sending message:", err)
				}
				continue
			}
			// Продолжайте с остальной логикой

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("1 минута", "1_minute"),
					tgbotapi.NewInlineKeyboardButtonData("1 час", "1_hour"),
					tgbotapi.NewInlineKeyboardButtonData("6 часов", "6_hours"),
				),
			)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите частоту обновлений:")
			msg.ReplyMarkup = keyboard
			_, err = bot.Send(msg)
			if err != nil {
				log.Println("Error sending message:", err)
			}

			err = updateUser(update.Message.Chat.ID, city, "initial")
			if err != nil {
				log.Println("Error inserting user:", err)
			}
			continue
		}

		if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			queryData := update.CallbackQuery.Data
			err := updateFrequency(chatID, queryData)
			if err != nil {
				log.Println("Error updating frequency:", err)
				msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при обновлении частоты. Пожалуйста, попробуйте позже.")
				_, err = bot.Send(msg)
				if err != nil {
					log.Println("Error sending error message:", err)
				}
				continue
			}

			sendWeather(bot, chatID, queryData)

			msg := tgbotapi.NewMessage(chatID, "Частота обновлений изменена!")
			_, err = bot.Send(msg)
			if err != nil {
				log.Println("Error sending message:", err)
			}
			continue
		}
	}
}
