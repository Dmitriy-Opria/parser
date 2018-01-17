package main

import (
	"bufio"
	"compress/gzip"
	"crypt_parser/config"
	"crypt_parser/model"
	"crypt_parser/register"
	"encoding/json"
	"fmt"
	"github.com/iizotop/texto/texto/utils"
	"golib/logs"
	"gopkg.in/telegram-bot-api.v4"
	"io/ioutil"
	"log"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	tooManyRequestsTimeOut time.Time
	telegramNotification   = make(chan string, 10)
)

const (
	Hour  = 0
	Day   = 1
	Week  = 2
	Month = 3
)

func main() {

	defer Recover()

	bot, err := tgbotapi.NewBotAPI("434988843:AAEk4HEnyxQT_RXGrAZYzSgYihlZlO-F3fA")
	if err != nil {
		log.Panic(err)
	}

	go worker()

	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for {

		select {
		case update := <-updates:

			if update.Message == nil {
				continue
			}
			if register.IsRegistered(update.Message.Chat.ID) {

				switch update.Message.Text {
				case "month":
					for _, msg := range getMsg(Month) {

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, msg)

						bot.Send(msg)
					}
				case "week":
					for _, msg := range getMsg(Week) {

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, msg)

						bot.Send(msg)
					}
				case "day":
					for _, msg := range getMsg(Day) {

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, msg)

						bot.Send(msg)
					}
				case "hour":
					for _, msg := range getMsg(Hour) {

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, msg)

						bot.Send(msg)
					}
				case "/get_stop":
					if register.RemoveUser(update.Message.Chat.ID) {

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы отписались от канала.")

						bot.Send(msg)
					}
				default:
					text := "Введите \"month\", \"week\", \"day\", \"hour\", для того что бы узнать обновления за месяц, неделю, день, час."

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)

					bot.Send(msg)
				}
			} else {
				switch update.Message.Text {
				case "/get_start":
					if register.SaveUser(update.Message.Chat.ID) {

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы успешно зарегистрированы.")

						bot.Send(msg)
					} else {

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Зарегистрироваться не удалось.")

						bot.Send(msg)
					}
				}
			}

		case text := <-telegramNotification:
			for id := range register.GetRegisteredList() {

				msg := tgbotapi.NewMessage(id.ID, text)

				bot.Send(msg)
			}
		}
	}
}

func worker() {
	for {
		conf := config.Get()
		saveNewCoins()
		time.Sleep(conf.TimePeriod)
	}
}

func saveNewCoins() {

	defer Recover()

	coinsToSave := make(map[string]model.Coin)

	currentCoinMap := getCurrentCoinsMap()

	savedCoinMap := getSavedCoinsMap()

	for coinName, coin := range currentCoinMap {

		if _, ok := savedCoinMap[coinName]; ok {
			continue
		}

		if coin.Name != "" {
			coin.InsertTime = time.Now()
			coinsToSave[coinName] = coin
		}
	}

	if len(coinsToSave) > 0 {
		saveUpdatedCoinsMap(coinsToSave)
		pushToTelegram(coinsToSave)
	}

}

func pushToTelegram(coins map[string]model.Coin) {
	if len(coins) == 0 {
		return
	}

	c := ""
	switch len(coins) {
	case 1:
		c = "монету"
	case 2:
		fallthrough
	case 3:
		fallthrough
	case 4:
		c = "монеты"
	default:
		c = "монет"
	}
	text := fmt.Sprintf("Было добавлено %d %s:\n", len(coins), c)

	for _, coin := range coins {
		text += fmt.Sprintf("\nНазвание: %s, Цена: %s", coin.Name, coin.Price)
	}
	telegramNotification <- text
}

func getCurrentCoinsMap() map[string]model.Coin {

	defer Recover()

	body := getResponseBody()

	res := make([]model.Coin, 256)

	coins := make(map[string]model.Coin, len(res))

	err := json.Unmarshal(body, &res)

	if err != nil {
		logs.Critical(fmt.Sprintf("Can`t unmarshal answer, error: %s", err.Error()))
	}

	for _, coin := range res {
		coins[coin.Name] = coin
	}
	return coins
}

func getSavedCoinsMap() (coins map[string]model.Coin) {

	defer Recover()

	coins = make(map[string]model.Coin, 0)
	conf := config.Get()

	file, err := os.Open(conf.BasePath)
	if err != nil {
		logs.Critical(fmt.Sprintf("Can`t open base file by path: %s, Error: %s", conf.BasePath, err.Error()))
		return
	}
	defer file.Close()

	reader := textproto.NewReader(bufio.NewReader(file))

	for {
		line, err := reader.ReadLine()
		if err != nil {
			break
		}
		coin := model.Coin{}
		json.Unmarshal([]byte(strings.TrimSuffix(line, ",")), &coin)
		coins[coin.Name] = coin
	}
	return coins
}

func getResponseBody() (body []byte) {

	defer Recover()

	conf := config.Get()

	request, err := http.NewRequest("GET", conf.SourceUrl, nil)

	if request.Response != nil {

		if request.Response.StatusCode == http.StatusTooManyRequests {
			telegramNotification <- "Превышен лимит запросов"
			tooManyRequestsTimeOut = time.Now().Add(time.Hour)

		}
	}

	if !time.Now().After(tooManyRequestsTimeOut) {
		return []byte{}
	}

	if err != nil {
		logs.Critical(fmt.Sprintf("Can`t make request, error: %s", err.Error()))
	}

	utils.SetBrowserHeaders(conf.SourceUrl, request.Header)

	client := http.Client{
		Timeout: conf.ResponseTimeOut,
	}

	response, err := client.Do(request)

	if err != nil {
		logs.Critical(fmt.Sprintf("Can`t do request, error: %s", err.Error()))
	}

	if response == nil {
		return
	}
	defer response.Body.Close()

	var bodyReader = response.Body

	if response.Header.Get("Content-Encoding") == "gzip" {
		bodyReader, err = gzip.NewReader(response.Body)
		defer bodyReader.Close()
	}

	body, err = ioutil.ReadAll(bodyReader)

	if err != nil {
		logs.Critical(fmt.Sprintf("Can`t get source response, error: %s", err.Error()))
	}

	return body
}

func saveUpdatedCoinsMap(coins map[string]model.Coin) {

	conf := config.Get()

	file, err := os.OpenFile(conf.BasePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		fmt.Printf("Can`t make json marshal, error: %s", err.Error())
	}
	defer file.Close()

	for _, coin := range coins {
		json.NewEncoder(file).Encode(coin)
		fmt.Println(coin)
	}
}

func getMsg(period int) (list []string) {
	tm := time.Time{}

	switch period {
	case Hour:
		tm = time.Now().Add(-1 * time.Hour)
	case Day:
		tm = time.Now().Add(-24 * time.Hour)
	case Week:
		tm = time.Now().Add(-7 * 24 * time.Hour)
	case Month:
		tm = time.Now().Add(-30 * 24 * time.Hour)
	}
	coinsMaps := make(map[string]model.Coin)

	for _, coin := range getSavedCoinsMap() {
		if coin.InsertTime.After(tm) {
			coinsMaps[coin.Name] = coin
		}
	}
	var msg string
	if len(coinsMaps) > 0 {
		switch period {
		case Hour:
			msg += "Новые монеты за час:"
		case Day:
			msg += "Новые монеты за день:"
		case Week:
			msg += "Новые монеты за неделю:"
		case Month:
			msg += "Новые монеты за месяц:"
		}

		counter := 1
		for _, coin := range coinsMaps {
			msg += fmt.Sprintf("\nНазвание: %s, Цена: %s", coin.Name, coin.Price)
			if counter%10 == 0 {
				list = append(list, msg)
				msg = ""
			}
			counter++
		}
		list = append(list, msg)
		msg = "  "
	}
	if msg == "" {
		switch period {
		case Hour:
			msg = "За последний час нет новых монет."
		case Day:
			msg = "За последний день нет новых монет."
		case Week:
			msg = "За последнюю неделю нет новых монет."
		case Month:
			msg = "За последний месяц нет новых монет."
		}
		list = append(list, msg)
	}

	return
}

func Recover() {

	if err := recover(); err != nil {

		pc, file, line, ok := runtime.Caller(4)

		if !ok {
			file = "?"
			line = 0
		}

		fnName := ""
		fn := runtime.FuncForPC(pc)

		if fn == nil {
			fnName = "?()"
		} else {
			dotName := filepath.Ext(fn.Name())
			fnName = strings.TrimLeft(dotName, ".") + "()"
		}

		var buf [10240]byte
		number := runtime.Stack(buf[:], false)

		debugStr := fmt.Sprintf("%s:%d %s: %s\n\n%s", file, line, fnName, err, buf[:number])

		logs.Critical(debugStr) // test
	}
}
