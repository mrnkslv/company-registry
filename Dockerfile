
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/company-registry ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/company-registry /company-registry
EXPOSE 8080
ENTRYPOINT ["/company-registry"]