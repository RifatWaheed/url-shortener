FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/snip .
# =================================================== #

FROM alpine:3.24
RUN adduser -D -u 10001 appuser
COPY --from=build /bin/snip /bin/snip
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/snip"]


