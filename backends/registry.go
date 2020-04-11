package backends

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/zeebo/errs"
)

type Creator func(ctx context.Context, url *url.URL) (Backend, error)

var (
	registryMtx sync.Mutex
	registry    = map[string]Creator{}
)

func Register(scheme string, c Creator) {
	registryMtx.Lock()
	defer registryMtx.Unlock()
	if _, exists := registry[scheme]; exists {
		panic(fmt.Sprintf("scheme %q already registered", scheme))
	}
	registry[scheme] = c
}

func Create(ctx context.Context, url *url.URL) (Backend, error) {
	registryMtx.Lock()
	defer registryMtx.Unlock()
	creator, exists := registry[url.Scheme]
	if !exists {
		return nil, errs.New("no backend registered with scheme %q", url.Scheme)
	}
	return creator(ctx, url)
}
