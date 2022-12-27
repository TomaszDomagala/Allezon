FROM golang:1.19 as builder

COPY ./ /go/src

WORKDIR /go/src/services/api

RUN go build -o /go/bin/api

# Two step compilation reduces size from 1.3 GB to ~20 MB.
FROM busybox

COPY --from=builder /go/bin/api /usr/bin/api

CMD ["/usr/bin/api"]