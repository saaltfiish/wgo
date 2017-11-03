package uranus

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
)

func Register(uranusHost, name, version, kind, host string, health *HealthInfo) {
	healthUrl := "NO"
	if health != nil {
		healthUrl = fmt.Sprintf("http://%s%s", host, health.Host)
	}
	jsonData := fmt.Sprintf("{\"name\": \"%s\", \"ver\": \"%s\", \"msg_url\": \"%s\", \"conn\": {\"host\": \"%s\", \"proto\": \"%s\"}}", name, version, healthUrl, host, kind)
	body := bytes.NewBuffer([]byte(jsonData))

	resp, err := http.Post(uranusHost+"/nodes", "application/json", body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		panic(errors.New("register error"))
	}
}
