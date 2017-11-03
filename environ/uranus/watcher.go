package uranus

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"encoding/json"
	"errors"
	"google.golang.org/grpc/naming"
)

type conn struct {
	Host  string `json:"host"`
	Name  string `json:"name"`
	Proto string `json:"proto"`
}

type node struct {
	Conn conn   `json:"conn"`
	Ver  string `json:"ver"`
}
type uranus struct {
	Add       []node `json:"add"`
	Del       []node `json:"del"`
	TimeStamp int64  `json:"time"`
}

type WatcherService struct {
	serviceName string
	version     string
	updateUrl   string
	timestamp   int64
}

func CreateWatcher(resolver *ResolverService) *WatcherService {
	return &WatcherService{
		serviceName: resolver.ServiceName,
		version:     resolver.Version,
		updateUrl:   resolver.HostUpdateUrl,
		timestamp:   0,
	}
}

func (obj *WatcherService) Close() {}

func (obj *WatcherService) Next() ([]*naming.Update, error) {
	uranus, err := obj.getUpdateNodes()
	if err != nil {
		return nil, nil
	}
	data := []*naming.Update{}
	for _, n := range uranus.Add {
		data = append(data, &naming.Update{Op: naming.Add, Addr: n.Conn.Host})
	}
	for _, n := range uranus.Del {
		data = append(data, &naming.Update{Op: naming.Delete, Addr: n.Conn.Host})
	}
	obj.timestamp = uranus.TimeStamp

	return data, nil
}

func (obj *WatcherService) getUpdateNodes() (*uranus, error) {

	resp, err := http.Get(fmt.Sprintf(obj.updateUrl, obj.serviceName, obj.version, obj.timestamp))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("timeout")
	}

	defer resp.Body.Close() //一定要关闭resp.Body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	uranus := &uranus{}
	err = json.Unmarshal(data, uranus)
	return uranus, err
}
