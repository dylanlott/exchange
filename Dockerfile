FROM golang:latest
WORKDIR /app 
ADD ./ /app
RUN go build -o main .
CMD ["/app/main"]
