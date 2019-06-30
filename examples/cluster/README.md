# Nano cluster example

## About this example



## How to run the example?

```shell
cd examples/cluster
go build

# run master server
./cluster master
./cluster chat --listen "127.0.0.1:34580"
./cluster gate --listen "127.0.0.1:34570" --gate-address "127.0.0.1:34590"
```

