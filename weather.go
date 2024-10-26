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

	// Получаем широту и долготу как строки и конвертируем в float64
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

func getWeatherFromOpenMeteo(latitude, longitude float64) (string, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&daily=temperature_2m_max,temperature_2m_min,precipitation_sum&timezone=Europe%%2FMoscow&past_days=0&forecast_days=1&hourly=temperature_2m,precipitation&current_weather=true&windspeed_10m=true&winddirection_10m=true&shortwave_radiation=true&precipitation_probability=true&cloudcover=true&visibility=true&surface_pressure=true&snowfall=true&showers=true&thunderstorm=true&temperature_10m=true&winddirection_10m_dominant=true&windgusts_10m=true&apparent_temperature=true", latitude, longitude)

	log.Printf("Запрос к Open-Meteo: %s", url) // Логирование URL

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

	dailyData := data["daily"].(map[string]interface{})
	temperatureMax := dailyData["temperature_2m_max"].([]interface{})[0].(float64)  // Изменено на float64
	temperatureMin := dailyData["temperature_2m_min"].([]interface{})[0].(float64)  // Изменено на float64
	precipitationSum := dailyData["precipitation_sum"].([]interface{})[0].(float64) // Изменено на float64

	message := fmt.Sprintf("🌤 Прогноз погоды:\n\n"+
		"🌡 Максимальная температура: %.2f°C\n"+
		"🌡 Минимальная температура: %.2f°C\n"+
		"💧 Количество осадков: %.2f мм\n", temperatureMax, temperatureMin, precipitationSum)
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
	weatherMessage, err := getWeatherFromOpenMeteo(coordinates.Latitude, coordinates.Longitude)
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
