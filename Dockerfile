# Build the manager binary
FROM golang:1.10 as builder

# Copy in the go src
WORKDIR /go/src/github.com/tangcong/codis-operator
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager github.com/tangcong/codis-operator/cmd/manager

# Copy the controller-manager into a thin image
FROM centos:latest
WORKDIR /root/
COPY --from=builder /go/src/github.com/tangcong/codis-operator/manager .
ENTRYPOINT ["./manager"]
