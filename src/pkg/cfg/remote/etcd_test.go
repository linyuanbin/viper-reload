package remote

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
	embedv3 "go.etcd.io/etcd/server/v3/embed"
)

type testRmoteProvider struct {
	provider  string
	endpoints string
	path      string
}

func (rp *testRmoteProvider) Provider() string {
	return rp.provider
}
func (rp *testRmoteProvider) Endpoint() string {
	return rp.endpoints
}
func (rp *testRmoteProvider) Path() string {
	return rp.path
}
func (rp *testRmoteProvider) SecretKeyring() string {
	return ""
}
func TestEtcd(t *testing.T) {
	urls := newEmbedURLs(2, t.TempDir())
	cfg := embedv3.NewConfig()
	cfg.Dir = t.TempDir()
	// disable etcd log
	cfg.LogLevel = "error"
	cfg.LPUrls = []url.URL{urls[0]}
	cfg.APUrls = []url.URL{urls[0]}
	cfg.LCUrls = []url.URL{urls[1]}
	cfg.ACUrls = []url.URL{urls[1]}
	cfg.InitialCluster = "default=" + urls[0].String()
	s, err := setupetcd(cfg)
	assert.Nil(t, err)
	defer s.Server.Stop()
	defer os.RemoveAll(t.TempDir())

	trp := &testRmoteProvider{
		provider:  "etcd",
		endpoints: urls[1].String(),
		path:      "/test/etcd",
	}

	vipertest := viper.New()
	vipertest.SetConfigType("yaml")
	err = vipertest.AddRemoteProvider(trp.Provider(), trp.Endpoint(), trp.Path())
	assert.Nil(t, err)

	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{trp.Endpoint()},
	})
	assert.Nil(t, err)

	_, err = client.Put(context.Background(), trp.Path(), "{name: a}")
	assert.Nil(t, err)
	defer client.Close()

	e := New()
	viper.RemoteConfig = e
	err = vipertest.WatchRemoteConfigOnChannel()
	assert.Nil(t, err)

	err = vipertest.ReadRemoteConfig()
	assert.Nil(t, err)
	assert.Equal(t, "a", vipertest.GetString("name"))

	exit := make(chan struct{})
	ch := e.GetChangeChannel(trp)

	teststrings := []struct {
		Content string
		Value   string
	}{
		{"name: 1", "1"},
		{"name: 2", "2"},
		{"name: hello", "hello"},
	}

	go func(tt *testing.T, c chan struct{}, vp *viper.Viper) {
		i := 0
		for {
			select {
			case <-c:
				err = vp.ReadRemoteConfig()
				assert.Nil(t, err)
				assert.Equal(tt, teststrings[i].Value, vp.GetString("name"))
				i++
			case <-exit:
				return
			}
		}
	}(t, ch, vipertest)
	for _, v := range teststrings {
		_, err := client.Put(context.Background(), trp.Path(), v.Content)
		assert.Nil(t, err)
		time.Sleep(time.Second)
	}
	time.Sleep(10 * time.Second)
	close(exit)
}

func setupetcd(cfg *embedv3.Config) (*embedv3.Etcd, error) {
	s, err := embedv3.StartEtcd(cfg)
	if err != nil {
		return nil, err
	}
	select {
	case <-s.Server.ReadyNotify():
		log.Println("Etcd embedded server is ready.")
		return s, nil
	case <-time.After(42 * time.Second):
		s.Server.Stop() // trigger a shutdown
		return nil, errors.New("Etcd embedded server took too long to start")
	case err := <-s.Err():
		return s, err
	}
}

func newEmbedURLs(n int, path string) (urls []url.URL) {
	scheme := "http"
	for i := 0; i < n; i++ {
		u, _ := url.Parse(fmt.Sprintf("%s://127.0.0.1:%d", scheme, 10000+i))
		urls = append(urls, *u)
	}
	return urls
}
