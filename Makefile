PWD = $(shell pwd)

run-etcd:
	etcd

run-cron:
	go run ./main.go start