# Awards Service 
Awards service handles the bookkeeping and journaling every time there is a gold/points transaction. Normally awards is a service for the other services to call.

## build

This project runs on gold...er go. You need golang installed on your system to run. Minimal version is **1.12** since this project uses `go mod`.  

You can go here for more information on go and installation.  
[golang resource](https://golang.org)  


To build you can type:  
`go build ./...`  

## tests

`go test ./... -v -covermode=count -coverprofile=coverage.out`  

or  

`make test`  
`make test-coverage`  

## generate-binary

`go build -a -o awards-go-img cmd/Main.go`  
  
or  
  
`make build`  

## create docker

`make docker`  

## run docker  

`make docker-run`  

## api docs  

Swagger docs can be seen by hitting the `/docs` endpoint of the running instance.
The file swagger.json can be found in `/api/swagger/spec`  

## Admin Dashboard

Dashboard can be accessed through `/dashboard` endpoint in the running instance.
User need to know the `SecretKey` used to generate the HMAC 
API Key.

## File structure 
(subject to change)

├── api  
│   └── swagger  
│       └── spec  
├── build  
│   └── azure  
├── cmd  
├── internal  
│   ├── config  
│   ├── connector  
│   ├── health  
│   ├── helpers  
│   ├── logger  
│   └── router  
└── migrations  

Further information see:  
1. [Golang Project Structure](https://tutorialedge.net/golang/go-project-structure-best-practices)  
2. [Golang standard project layout ](https://github.com/golang-standards/project-layout)  

