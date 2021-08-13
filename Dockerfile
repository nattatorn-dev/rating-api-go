FROM golang:1.16-alpine AS builder
LABEL maintainer="Nattatorn Yucharoen <nattatorn.dev@gmail.com>"
WORKDIR /src
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:3.14.1  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /src/main .
COPY .env .
CMD ["./main"] 
