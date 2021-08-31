FROM golang:1.16 as build
WORKDIR /go/src/registration-service/
COPY . .
RUN go build -o /build/rs_server -tags interpreted_hardware /go/src/registration-service

# The production image, based on google distroless
# See https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/base-debian10

COPY --from=build /build/rs_server /

CMD ["/rs_server"]