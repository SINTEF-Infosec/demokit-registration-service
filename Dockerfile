FROM golang:1.16 as build
WORKDIR /go/src/registration-service/
COPY . .
RUN go build -o /build/registration-service /go/src/registration-service

# The production image, based on google distroless
# See https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/base-debian10

COPY --from=build /build/registration-service /

CMD ["/registration-service"]
