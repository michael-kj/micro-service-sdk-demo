package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/namespace"
	"google.golang.org/grpc"
	"log"
	"strconv"
	"sync"
	"time"
)

var NotExistError = errors.New("NotExistErr")

type Client interface {
	Connect() error
	Watch()
	Load() error
}

type Storage interface {
	Set(key string, value []byte)
	GetString(key string) (string, error)
	GetInt(key string) (*int, error)
	GetBytes(key string) ([]byte, error)
	GetObject(key string, target interface{}) error
	Init() error
}

type MapStorage struct {
	d    map[string][]byte
	lock sync.RWMutex
}

func (m *MapStorage) Set(key string, value []byte) {
	m.lock.Lock()
	m.d[key] = value
	m.lock.Unlock()

}
func (m *MapStorage) GetString(key string) (string, error) {
	data, err := m.GetBytes(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (m *MapStorage) GetInt(key string) (*int, error) {
	data, err := m.GetString(key)

	if err != nil {
		return nil, err
	}
	i, err := strconv.Atoi(data)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (m *MapStorage) GetBytes(key string) ([]byte, error) {
	m.lock.RLock()

	value, ok := m.d[key]

	if !ok {
		return value, NotExistError
	}
	m.lock.RUnlock()

	return value, nil
}

func (m *MapStorage) GetObject(key string, target interface{}) error {
	data, err := m.GetBytes(key)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		return err
	}
	return nil
}

func (m *MapStorage) Init() error {
	if m.d == nil {
		m.d = make(map[string][]byte)
	}
	return nil
}

type EtcdClient struct {
	c                *clientv3.Client
	ProjectNameSpace string
	storage          Storage
}

func (e *EtcdClient) Connect() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            []string{"localhost:2379"},
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    5 * time.Second,
		DialKeepAliveTimeout: 5 * time.Second,
		DialOptions:          []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	e.c = cli
	return nil

}

func (e *EtcdClient) Watch() {

	go e.watch()
}

func (e *EtcdClient) Load() error {
	//ctx,_:=context.WithTimeout(context.Background(),time.Second)
	ctx := context.Background()
	resp, err := e.c.Get(ctx, fmt.Sprintf("%s/", e.ProjectNameSpace), clientv3.WithPrefix())
	if err != nil {
		log.Printf("err:%s \n", err)
		return err
	}
	for _, ev := range resp.Kvs {
		e.Set(string(ev.Key), ev.Value)
		log.Printf("%s : %s\n", ev.Key, ev.Value)
	}
	return nil
}

func (e *EtcdClient) watch() {
	w := namespace.NewWatcher(e.c.Watcher, fmt.Sprintf("%s/", e.ProjectNameSpace)).Watch(clientv3.WithRequireLeader(context.Background()), "", clientv3.WithPrefix())
	for wresp := range w {
		log.Printf("closed:%t\n", wresp.Canceled)
		for _, ev := range wresp.Events {
			e.storage.Set(string(ev.Kv.Key), ev.Kv.Value)
			log.Printf("%s %q : %q, modi:%t create:%t \n", ev.Type, ev.Kv.Key, ev.Kv.Value, ev.IsModify(), ev.IsCreate())
		}
	}
}
func (e *EtcdClient) SetStorage(s Storage) error {
	err := s.Init()
	if err != nil {
		return err
	}
	e.storage = s
	return nil
}

func (e *EtcdClient) Set(key string, value []byte) {

	e.storage.Set(key, value)

}
func (e *EtcdClient) GetString(key string) (string, error) {

	return e.storage.GetString(key)
}

func (e *EtcdClient) GetInt(key string) (*int, error) {

	return e.storage.GetInt(key)
}

func (e *EtcdClient) GetObject(key string, target interface{}) error {

	return e.storage.GetObject(key, target)
}
