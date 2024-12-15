package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	elector "github.com/go-co-op/gocron-etcd-elector"
	"github.com/go-co-op/gocron/v2"

	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cli := &cobra.Command{}

	cli.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "start cron job",
		Long:  "start cron job",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := elector.Config{
				Endpoints:   []string{"http://localhost:2379"},
				DialTimeout: 3 * time.Second,
			}

			el, err := elector.NewElector(cmd.Context(), cfg, elector.WithTTL(10))
			if err != nil {
				return err
			}

			log.Printf("Elector ID: %v", el.GetID())

			go func() {
				if err := el.Start("/distributed-cron-job/elector"); err == elector.ErrClosed {
					log.Fatalf("Elector closed: %v", err)
				}
			}()

			sh, err := gocron.NewScheduler(gocron.WithDistributedElector(el))
			if err != nil {
				return err
			}

			if _, err := sh.NewJob(
				gocron.DurationJob(1*time.Second),
				gocron.NewTask(
					func() {
						log.Printf("Current Leader: %v", el.GetLeaderID())
					},
				),
			); err != nil {
				return err
			}

			sh.Start()

			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
			<-c

			return nil
		},
	})

	return cli
}
