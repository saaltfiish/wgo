package uranus

import (
	"fmt"

	"io/ioutil"
	"net/http"

	"github.com/bitly/go-simplejson"
)

func GetConfigContent(addr, name, version, env string) (string, string, error) {

	url := fmt.Sprintf("http://%s/services?name=%s&ver=%s&env=%s", addr, name, version, env)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}

	defer resp.Body.Close() //一定要关闭resp.Body
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	jsonData, err := simplejson.NewJson(d)
	if err != nil {
		return "", "", err
	}
	sid, err := jsonData.Get("list").GetIndex(0).Get("id").String()
	if err != nil {
		return "", "", err
	}
	content, err := jsonData.Get("list").GetIndex(0).Get("config").Map()
	if err != nil {
		return "", "", err
	}

	return sid, content["content"].(string), nil
}
