# nano
定制版 nano

## dev
### proto gen
* `protoc -I=./proto/chat_proto/ --go_out=./proto/chat_proto/ ./proto/chat_proto/chat.proto`
* 或者借助脚本：`./build.sh chat_proto/v1`
