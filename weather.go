package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var userTimers = make(map[int64]*time.Timer)

func getCoordinates(city string) (*Coordinates, error) {
	cityEncoded := url.QueryEscape(city)
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1", cityEncoded)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	var data []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("city not found")
	}

	latStr := data[0]["lat"].(string)
	lonStr := data[0]["lon"].(string)
	latitude, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing latitude: %s", err)
	}
	longitude, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing longitude: %s", err)
	}

	coordinates := Coordinates{
		Latitude:  latitude,
		Longitude: longitude,
	}
	return &coordinates, nil
}

func getWeatherFromOpenMeteo(latitude, longitude float64, city string) (string, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=temperature_2m_max,temperature_2m_min,precipitation_sum&timezone=Europe%%2FMoscow&current_weather=true", latitude, longitude)

	// log.Printf("Запрос к Open-Meteo: %s", url) 

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	currentWeather := data["current_weather"].(map[string]interface{})
	temperature := currentWeather["temperature"].(float64)
	windSpeed := currentWeather["windspeed"].(float64)

	dailyData := data["daily"].(map[string]interface{})
	precipitationSum := dailyData["precipitation_sum"].([]interface{})[0].(float64)


	message := fmt.Sprintf("🌤 Прогноз погоды для города: %s\n\n"+
		"🌡 Текущая температура: %.1f°C\n"+
		"💧 Количество осадков: %.1f мм\n"+
		"🌬 Скорость ветра: %.1f м/с\n", city, temperature, precipitationSum, windSpeed)

	return message, nil
}

func isValidCity(city string) (bool, error) {
	_, err := getCoordinates(city)
	if err != nil {
		return false, err
	}
	return true, nil
}

func sendWeather(bot *tgbotapi.BotAPI, chatID int64, frequency string) {
	city, err := getCityByChatID(chatID)
	if err != nil {
		log.Println("Error getting city from database:", err)
		return
	}
	if timer, exists := userTimers[chatID]; exists {
		timer.Stop()
	}
	coordinates, err := getCoordinates(city)
	if err != nil {
		log.Printf("Ошибка при получении координат для города '%s': %s\n", city, err.Error())
		return
	}
	weatherMessage, err := getWeatherFromOpenMeteo(coordinates.Latitude, coordinates.Longitude, city)
	if err != nil {
		log.Printf("Ошибка при получении погоды для города '%s': %s\n", city, err.Error())
		return
	}
	msg := tgbotapi.NewMessage(chatID, weatherMessage)
	_, err = bot.Send(msg)
	if err != nil {
		log.Println("Error sending message:", err)
	}
	var duration time.Duration
	switch frequency {
	case "1_minute":
		duration = 1 * time.Minute
	case "1_hour":
		duration = 1 * time.Hour
	case "6_hours":
		duration = 6 * time.Hour
	}
	timer := time.AfterFunc(duration, func() {
		sendWeather(bot, chatID, frequency)
	})
	userTimers[chatID] = timer
}
