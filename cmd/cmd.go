package cmd

import (
	"context"
	"distributed-cron-job/internal/elector"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func New() *cobra.Command {
	cli := &cobra.Command{}

	cli.AddCommand(&cobra.Command{
		Use:   "run-elector",
		Short: "run a distributed cron job using leader election",
		Long:  "run a distributed cron job using leader election",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := clientv3.Config{
				Endpoints:   []string{"http://localhost:2379"},
				DialTimeout: 5 * time.Second,
			}

			client, err := clientv3.New(config)
			if err != nil {
				return err
			}
			defer client.Close()

			session, err := concurrency.NewSession(client, concurrency.WithTTL(10))
			if err != nil {
				return err
			}
			defer session.Close()

			election := concurrency.NewElection(session, "/distributed-cron-job/elector")
			elector := elector.NewElector(election)

			log.Printf("starting elector: %s", elector.GetID().String())

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			errCh := make(chan error, 1)
			ticker := time.NewTicker(time.Second)

			go elector.CampaignLoop(ctx, ticker, errCh)

			sh, err := gocron.NewScheduler(gocron.WithDistributedElector(elector))
			if err != nil {
				return err
			}

			if _, err := sh.NewJob(
				gocron.DurationJob(5*time.Second),
				gocron.NewTask(
					func() {
						log.Printf("current leader: %v", elector.GetID())
					},
				),
			); err != nil {
				return err
			}

			sh.Start()
			defer sh.Shutdown()

			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

			select {
			case <-c:
				return nil
			case err := <-errCh:
				log.Printf("error: %v", err)
				return nil
			}
		},
	})

	return cli
}
