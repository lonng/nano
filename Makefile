### Makefile for nano
GO        := GO111MODULE=on go
GOBUILD   := GO111MODULE=on CGO_ENABLED=0 $(GO) build

ARCH      := "`uname -s`"
LINUX     := "Linux"
MAC       := "Darwin"


.PHONY: test proto

test:
	go test -v ./...

proto:
	@cd ./cluster/clusterpb/proto/ && protoc --go_out=plugins=grpc:../ *.proto

.PHONY: up
up:
	git add .
	git commit -am "update"
	git pull origin master
	git push origin master
	@echo "\n game update 发布中..."