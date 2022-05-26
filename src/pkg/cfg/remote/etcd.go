package remote

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	_ "google.golang.org/grpc/naming"
)

var (
	_            Config = &etcd{}
	providerOnce sync.Once
	Provider     Config
)

type Config interface {
	Get(rp viper.RemoteProvider) (io.Reader, error)
	Watch(rp viper.RemoteProvider) (io.Reader, error)
	WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool)

	GetChangeChannel(rp viper.RemoteProvider) chan struct{}
}

type etcd struct {
	rw sync.RWMutex
	// remote_key : watchChan
	watchChanMap map[string]chan struct{}
}

func (e *etcd) Get(rp viper.RemoteProvider) (io.Reader, error) {
	return e.get(rp)
}

func (e *etcd) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	return e.get(rp)
}

func (e *etcd) WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	rr := make(chan *viper.RemoteResponse)
	stop := make(chan bool)
	go func() {
		client, err := clientv3.New(clientv3.Config{
			Endpoints: []string{rp.Endpoint()},
		})

		if err != nil {
			return
		}
		defer client.Close()
		ch := client.Watch(context.Background(), rp.Path())
		notify := e.GetChangeChannel(rp)
		for {
			select {
			case <-stop:
				return
			case res := <-ch:
				for _, event := range res.Events {
					rr <- &viper.RemoteResponse{
						Value: event.Kv.Value,
					}
					notify <- struct{}{}
				}
			}
		}
	}()
	return rr, stop
}

func (e *etcd) get(rp viper.RemoteProvider) (io.Reader, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{rp.Endpoint()},
	})

	if err != nil {
		return nil, err
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	resp, err := client.Get(ctx, rp.Path())
	cancel()
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(resp.Kvs[0].Value), nil
}

func (e *etcd) GetChangeChannel(rp viper.RemoteProvider) chan struct{} {
	e.rw.Lock()
	defer e.rw.Unlock()
	ch, ok := e.watchChanMap[rp.Path()]
	if !ok {
		ch = make(chan struct{}, 1)
		e.watchChanMap[rp.Path()] = ch
	}
	return ch
}

func New() Config {
	providerOnce.Do(
		func() {
			Provider = &etcd{
				watchChanMap: make(map[string]chan struct{}),
			}
		})
	return Provider
}
