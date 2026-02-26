FROM golang:alpine AS build
WORKDIR /src

# RUN apk add gcc musl-dev

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY --parents main.go storage routes ./

ENV GOCACHE=/gocache
RUN --mount=type=cache,target="${GOCACHE}" go build -v -o /box

# #2 TODO: вернуть, когда появятся тесты
# Run the tests in the container
# FROM build AS run-test-stage
# RUN go test -v


FROM alpine AS main
WORKDIR /

COPY --from=build /box /box

ENV GIN_MODE=release
ENV PORT=8080

ENTRYPOINT ["/box"]