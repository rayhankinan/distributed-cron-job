PWD = $(shell pwd)

run-etcd:
	etcd

run-with-elector:
	go run ./main.go run-with-elector

run-with-lock:
	go run ./main.go run-with-lock