package main

import "distributed-cron-job/cmd"

func main() {
	cli := cmd.New()
	_ = cli.Execute()
}
