FROM golang AS builder

RUN apt-get install git -y

ENV GO111MODULE=on

WORKDIR /
COPY . /

RUN CGO_ENABLED=0 go build main.go
RUN go get golang.org/x/tools/cmd/benchcmp

FROM golang

RUN apt-get install git -y

COPY --from=builder /main /main
COPY --from=builder ${GOPATH}/bin/benchcmp ${GOPATH}/bin/



ENTRYPOINT ["/main"]

