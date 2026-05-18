ARG GO_IMAGE=golang:1.26.2-alpine

FROM ${GO_IMAGE} AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP=agent
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${APP}

FROM scratch
COPY --from=build /out/app /app

EXPOSE 8080 9100
ENTRYPOINT ["/app"]
