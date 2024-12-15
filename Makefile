PWD = $(shell pwd)

run-etcd:
	etcd

run-elector:
	go run ./main.go run-elector