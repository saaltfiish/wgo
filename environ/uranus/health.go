package uranus

import "net/http"

type CheckHealthFunc func() bool

type HealthInfo struct {
	Host string
	RetCode int
	CheckFunc CheckHealthFunc
}

func (obj *HealthInfo) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if obj.CheckFunc(){
		w.WriteHeader(obj.RetCode)
	}else{
		w.WriteHeader(500)
	}
}

