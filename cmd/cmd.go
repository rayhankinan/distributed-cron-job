package cmd

import (
	"context"
	"distributed-cron-job/internal/elector"
	"distributed-cron-job/internal/locker"
	"log"
	"os"
	"os/signal"
	"strconv"
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
		Use:   "run-with-elector",
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

			e := elector.NewElector(session, "/distributed-cron-job/elector")
			log.Printf("starting elector: %s", e.GetID().String())

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			errCh := make(chan error, 1)
			ticker := time.NewTicker(time.Second)

			go e.CampaignLoop(ctx, ticker, errCh)

			sh, err := gocron.NewScheduler(gocron.WithDistributedElector(e))
			if err != nil {
				return err
			}

			if _, err := sh.NewJob(
				gocron.DurationJob(time.Second),
				gocron.NewTask(
					func() {
						log.Printf("current leader: %v", e.GetID())
					},
				),
				gocron.WithName("job-with-elector"),
			); err != nil {
				return err
			}

			sh.Start()
			defer sh.Shutdown()

			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

			select {
			case s := <-c:
				log.Printf("received signal: %v", s)
				return nil
			case err := <-errCh:
				log.Printf("error: %v", err)
				return err
			}
		},
	})

	cli.AddCommand(&cobra.Command{
		Use:   "run-with-lock",
		Short: "run a distributed cron job using distributed lock",
		Long:  "run a distributed cron job using distributed lock",
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

			l := locker.NewLocker(session)

			log.Printf("starting locker")

			sh, err := gocron.NewScheduler(gocron.WithDistributedLocker(l))
			if err != nil {
				return err
			}

			if _, err := sh.NewJob(
				gocron.DurationJob(time.Second),
				gocron.NewTask(
					func() {
						rawData, err := os.ReadFile("./resource/test.counter")
						if err != nil {
							log.Printf("error reading file: %v", err)
							return
						}

						data, err := strconv.Atoi(string(rawData))
						if err != nil {
							log.Printf("error converting data: %v", err)
							return
						}

						log.Printf("current counter: %d", data)

						data++

						if err := os.WriteFile("./resource/test.counter", []byte(strconv.Itoa(data)), 0740); err != nil {
							log.Printf("error writing file: %v", err)
							return
						}
					},
				),
				gocron.WithName("job-with-resource"),
			); err != nil {
				return err
			}

			sh.Start()
			defer sh.Shutdown()

			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

			s := <-c
			log.Printf("received signal: %v", s)

			return nil
		},
	})

	return cli
}
