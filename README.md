# go-wiki

Simple single-user wiki system.

## What is this good for?

My personal notes ;)

## How to

```
go get github.com/reekoheek/go-wiki
```

## Compile bindata
```
go get -u github.com/jteeuwen/go-bindata/...

rm ./bindata.go
go-bindata -ignore '[.]DS_Store' www/... templates/...
```