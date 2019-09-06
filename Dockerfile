FROM node:alpine AS webapp
WORKDIR /app
COPY ./web/exchange /app
RUN npm install && npm run build

FROM golang:latest
WORKDIR /app 
COPY ./ /app
COPY --from=webapp /app/dist ./web/exchange/
RUN go build -o main .
EXPOSE 9000
CMD ["/app/main"]
