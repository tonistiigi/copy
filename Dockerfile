from golang:1.7-alpine AS main
copy . .
run go build -o /copy -ldflags "-s -w" main.go

from lalyos/upx AS upx
copy --from=main /copy /copy
run ["upx", "/copy"]

from alpine AS cp
run apk add --no-cache wget
run wget http://s.minos.io/archive/bifrost/x86_64/coreutils-7.6-5.tar.gz
run tar xvf coreutils-7.6-5.tar.gz -C /

from scratch
copy --from=upx /copy /bin/copy
copy --from=cp /bin/cp /bin/cp
entrypoint ["/bin/copy"]