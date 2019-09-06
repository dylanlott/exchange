FROM node:alpine AS webapp
WORKDIR /app
COPY ./web/exchange /app
RUN npm install && npm run build

FROM golang:latest
WORKDIR /app 
COPY ./ /app
RUN mkdir -p /web/exchange/dist
COPY --from=webapp /app/dist ./web/exchange/dist/
RUN go build -o main .
EXPOSE 9000
CMD ["/app/main"]
