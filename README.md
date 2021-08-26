# Hyperjump Wallet Service

Hyperwallet is a generic bookkeeping and journaling management service. It implements the [acccore](https://github.com/hyperjumptech/acccore) accounting library. The service is intended for any applications where double-entry book keeping is required.

## building  

This project is built on golang. You need golang installed on your system to run. Minimal version is **1.16** since this project uses `go mod` and resource `embedding`.  

You can click here for the [golang resource](https://golang.org)   and for more information on go and installations.

To build you can type:  
`go build ./...`  

## testing

`go test ./... -v -covermode=count -coverprofile=coverage.out`  

or  

`make test`  
`make test-coverage`  

## binary generation

`go build -a -o wallet-go-img cmd/Main.go`  
  
or  
  
`make build`  

## docker generation

`make docker`  

## running docker  

`make docker-run`  

## api docs  

Open API sepecificaations can be seen by hitting the `/docs` endpoint of the running instance.
The file swagger.json can be found in `/static/api/spec`  

## Admin Dashboard

Dashboard can be accessed through `/dashboard` endpoint in the running instance.
User need to know the `SecretKey` used to generate the HMAC API Key.

## File structure  

├── build  
│   ├── azure  
│   ├── github  
│   └── docker  
├── cmd  
├── errors  
├── internal  
│   ├── accounting  
│   ├── config  
│   ├── connector  
│   ├── contextkeys  
│   ├── health  
│   ├── helpers  
│   ├── logger  
│   ├── middlewares  
│   └── router  
├── migrations  
├── static  
│   ├── api  
│   ├── dashboard  
│   └── mime  

## Further information see:  

1. [Golang Project Structure](https://tutorialedge.net/golang/go-project-structure-best-practices)  
2. [Golang standard project layout ](https://github.com/golang-standards/project-layout)  
