package register

import (
	"bufio"
	"crypt_parser/config"
	"encoding/json"
	"fmt"
	"golib/logs"
	"net/textproto"
	"os"
	"strings"
	"io/ioutil"
)

type RegisteredID struct {
	ID int64 `json:"id"`
}

var (
	list = map[RegisteredID]struct{}{}
)

func init() {
	list = GetRegisteredList()
}

func GetRegisteredList() (IDs map[RegisteredID]struct{}) {
	IDs = make(map[RegisteredID]struct{}, 5)
	conf := config.Get()

	file, err := os.Open(conf.RegisterPath)
	if err != nil {
		logs.Critical(fmt.Sprintf("Can`t open register file by path: %s, Error: %s", conf.RegisterPath, err.Error()))
		return
	}
	defer file.Close()

	reader := textproto.NewReader(bufio.NewReader(file))

	for {
		line, err := reader.ReadLine()
		if err != nil {
			break
		}
		ID := RegisteredID{}
		json.Unmarshal([]byte(strings.TrimSuffix(line, ",")), &ID)
		IDs[ID] = struct{}{}
	}
	return IDs

}
func SaveUser(id int64) bool {

	conf := config.Get()

	file, err := os.OpenFile(conf.RegisterPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		fmt.Printf("Can`t make json marshal, error: %s", err.Error())
	}
	defer file.Close()

	registeredID := RegisteredID{
		ID: id,
	}
	list[registeredID] = struct{}{}
	err = json.NewEncoder(file).Encode(registeredID)
	if err != nil {
		return false
	}
	return true
}

func IsRegistered(id int64) (ok bool) {
	registerID := RegisteredID{
		ID: id,
	}
	_, ok = list[registerID]

	return ok
}

func RemoveUser(id int64) (ok bool) {
	removalID := RegisteredID{
		ID: id,
	}

	delete(list,removalID)

	conf := config.Get()

	buf := ""

	if err := ioutil.WriteFile(conf.RegisterPath, []byte(buf), os.ModePerm); err != nil {
		return
	}

	for k := range list {
		SaveUser(k.ID)
	}

	return true
}
