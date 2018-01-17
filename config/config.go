package config

import (
	"encoding/json"
	"fmt"
	"golib/logs"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type (
	Data struct {
		SourceUrl          string `json:"source_url"`
		TimePeriodI64      int64  `json:"reload_period"`
		ResponseTimeOutI64 int64  `json:"response_time_out"`
		BasePath           string `json:"base_path"`
		TimePeriod         time.Duration
		ResponseTimeOut    time.Duration
		RegisterPath       string `json:"register_path"`
	}

	SMTP struct {
	}
)

const (
	DefaultConfigName   = "stat.conf"
	DefaultTimeout      = 60
	DefaultTimePeriod   = 1
	DefaulBaseFileName  = "base.txt"
	DefaultRegisterPath = "register.txt"
)

var (
	configName       = DefaultConfigName
	globalConfigData atomic.Value
	once             sync.Once
)

func init() {
	globalConfigData.Store(&Data{})
	load(true)
}

func load(first bool) {
	Data := new(Data)
	filename := configName

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		logs.Critical(fmt.Sprintf("Config file \"%s\" not found", filename))
		if first {
			os.Exit(1)
		} else {
			return
		}
	}

	err = json.Unmarshal(data, &Data)
	if err != nil {
		logs.Critical(fmt.Sprintf("Fail to parse config file \"%s\"", filename))
		if first {
			os.Exit(1)
		} else {
			return
		}
	}
	if Data.TimePeriodI64 == 0 {
		Data.TimePeriodI64 = DefaultTimePeriod
	}

	if Data.ResponseTimeOutI64 == 0 {
		Data.ResponseTimeOutI64 = DefaultTimeout
	}

	if Data.BasePath == "" {
		Data.BasePath = DefaulBaseFileName
	}

	if Data.RegisterPath == "" {
		Data.RegisterPath = DefaultRegisterPath
	}

	Data.TimePeriod = time.Duration(time.Duration(Data.TimePeriodI64) * time.Minute)
	Data.ResponseTimeOut = time.Duration(time.Duration(Data.ResponseTimeOutI64) * time.Second)

	globalConfigData.Store(Data)

	go once.Do(checkUpdate)

}

func checkUpdate() {

	info, err := os.Stat(configName)
	if err != nil {
		logs.Critical(err.Error())
		return
	}

	startTime := info.ModTime()

	for {

		time.Sleep(1 * time.Minute)

		if info, err = os.Stat(configName); err == nil {

			if info.ModTime().Sub(startTime) > 0 {

				load(false)
				startTime = info.ModTime()
			}
		}
	}
}

func Get() *Data {

	return globalConfigData.Load().(*Data)
}
