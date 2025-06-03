FROM golang:1.24.3-alpine3.21
WORKDIR /usr/broker

COPY . . 
RUN go build -o brokerApp ./cmd

ENV HOME=/root 

CMD ["./brokerApp"]