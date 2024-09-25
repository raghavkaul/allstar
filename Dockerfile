FROM golang:1.23.0@sha256:8e529b64d382182bb84f201dea3c72118f6ae9bc01d27190ffc5a54acf0091d2 AS base
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . ./

FROM base AS build
RUN go build ./cmd/allstar
RUN ls

FROM cgr.dev/chainguard/wolfi-base@sha256:7574456f268bc839ac78828865087c04a4297ca226b0eb5d051d4222e7690081
COPY --from=build /src/allstar /
# git binary is needed for go-git
RUN apk add git
ENTRYPOINT [ "/allstar" ]