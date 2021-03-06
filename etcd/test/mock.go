package test

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.etcd.io/etcd/v3/client"
)

type KeysAPIMock struct {
	mock.Mock
}

func (a *KeysAPIMock) Get(ctx context.Context, key string, opts *client.GetOptions) (*client.Response, error) {
	args := a.Called(ctx, key, opts)
	if r, ok := args.Get(0).(*client.Response); ok {
		return r, args.Error(1)
	}

	return nil, args.Error(1)
}

func (a *KeysAPIMock) Set(ctx context.Context, key, value string, opts *client.SetOptions) (*client.Response, error) {
	args := a.Called(ctx, key, value, opts)
	if r, ok := args.Get(0).(*client.Response); ok {
		return r, args.Error(1)
	}

	return nil, args.Error(1)
}

func (a *KeysAPIMock) Delete(ctx context.Context, key string, opts *client.DeleteOptions) (*client.Response, error) {
	return nil, nil
}

func (a *KeysAPIMock) Create(ctx context.Context, key, value string) (*client.Response, error) {
	panic("not implemented")
}

func (a *KeysAPIMock) CreateInOrder(ctx context.Context, dir, value string, opts *client.CreateInOrderOptions) (*client.Response, error) {
	panic("not implemented")
}

func (a *KeysAPIMock) Update(ctx context.Context, key, value string) (*client.Response, error) {
	panic("not implemented")
}

func (a *KeysAPIMock) Watcher(key string, opts *client.WatcherOptions) client.Watcher {
	panic("not implemented")
}
