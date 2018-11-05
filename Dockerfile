FROM golang:latest as build-env

WORKDIR /go/src/app
#ADD main.go /go/src/app

#RUN go-wrapper download   # "go get -d -v ./..."
#RUN go-wrapper install

RUN go get -v -u github.com/shadowkrusha/snogo/cmd/snogo
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build 
#RUN go build /go/

FROM gcr.io/distroless/base:latest
COPY --from=build-env /go/bin/snogo /
ENTRYPOINT ["/snogo"]
