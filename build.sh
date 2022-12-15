#!/bin/bash

CURRENT_PATH=$(cd `dirname $0` && pwd)

if [[ $# -lt 1 ]]; then
    echo "usage: $0 helloworld/v1"
    exit 1
fi

if [ -d "${CURRENT_PATH}/proto/$1" ];then
  rm -rf "${CURRENT_PATH}/golang/$1"
  mkdir -p "${CURRENT_PATH}/golang/$1"
  cp -rf ${CURRENT_PATH}/proto/$1/*.proto "${CURRENT_PATH}/golang/$1/"
  cd "${CURRENT_PATH}/golang/$1" || exit 1
  # shellcheck disable=SC2045
  for file in $(ls *.proto)
  do
    if [[ "${file}" = *model* ]];then
      #kratos proto client -p "${CURRENT_PATH}/proto" "${file}"
      protoc --proto_path=. \
	       --proto_path=${CURRENT_PATH}/proto \
	       --proto_path=${CURRENT_PATH}/proto/third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
           --go-errors_out=paths=source_relative:. \
 	       --openapi_out=naming=proto,paths=source_relative:. \
           "${file}"
    fi
  done

  # shellcheck disable=SC2045
  for file in $(ls *.proto)
  do
    if [[ "${file}" != *model* ]];then
      #kratos proto client -p "${CURRENT_PATH}/proto" "${file}"
      protoc --proto_path=. \
	       --proto_path=${CURRENT_PATH}/proto \
	       --proto_path=${CURRENT_PATH}/proto/third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
           --go-errors_out=paths=source_relative:. \
 	       --openapi_out=naming=proto,paths=source_relative:. \
           "${file}"
    fi
  done

  if [[ "$1" != *common* ]];then
    mkdir -p "service"
    rm -rf "service/"*.go
    # shellcheck disable=SC2045
    for file in $(ls *.proto)
    do
      kratos proto server "${file}" -t "service"
    done
  fi
  rm -rf *.proto
else
  echo "文件不存在"
  exit 1
fi




