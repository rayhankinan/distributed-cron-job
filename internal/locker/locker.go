package locker

import (
	"context"

	"github.com/go-co-op/gocron/v2"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var _ gocron.Locker = (*Locker)(nil)

type Locker struct {
	session *concurrency.Session
}

func (l *Locker) Lock(ctx context.Context, key string) (gocron.Lock, error) {
	mu := concurrency.NewMutex(l.session, key)

	if err := mu.Lock(ctx); err != nil {
		return nil, err
	}

	return mu, nil
}

func NewLocker(session *concurrency.Session) *Locker {
	return &Locker{session: session}
}
