package uranus

import (
	"fmt"

	"google.golang.org/grpc/naming"
)

type ResolverService struct {
	ServiceName   string
	Version       string
	HostUpdateUrl string
}

func CreateResolverService(serviceName string, version string, updateUrl string) *ResolverService {

	return &ResolverService{
		ServiceName:   serviceName,
		Version:       version,
		HostUpdateUrl: updateUrl,
	}
}

func (obj *ResolverService) Resolve(serviceName string) (naming.Watcher, error) {
	fmt.Println("get  ", serviceName)

	return CreateWatcher(obj), nil
}
