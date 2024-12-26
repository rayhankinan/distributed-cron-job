package elector

import (
	"context"
	"errors"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var _ gocron.Elector = (*Elector)(nil)

type Elector struct {
	id       uuid.UUID
	election *concurrency.Election
}

func (e *Elector) GetID() uuid.UUID {
	return e.id
}

func (e *Elector) CampaignLoop(ctx context.Context, ticker *time.Ticker, errCh chan<- error) {
	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
			if err := e.election.Campaign(ctx, e.id.String()); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func (e *Elector) IsLeader(ctx context.Context) error {
	response, err := e.election.Leader(ctx)
	if err != nil {
		return err
	}

	for _, kv := range response.Kvs {
		value, err := uuid.ParseBytes(kv.Value)
		if err != nil {
			continue
		}

		if value != e.id {
			continue
		}

		return nil
	}

	return errors.New("the elector is not leader")
}

func NewElector(session *concurrency.Session, pfx string) *Elector {
	return &Elector{
		id:       uuid.New(),
		election: concurrency.NewElection(session, pfx),
	}
}
