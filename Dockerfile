FROM alpine:latest

# Manifest dependancies
RUN apk update && apk add --no-cache go git

# Sets up manifestuser as user 1001 ready to execute manifest
RUN adduser -D -u 1001 manifestuser
USER manifestuser:1001
WORKDIR /manifest
COPY ./ ./
RUN go install -buildvcs=false ./cmd/manifest
ENV PATH="/home/manifestuser/go/bin:${PATH}"

# /app is where the repository to be checked should be mounted
WORKDIR /app
RUN git config --global --add safe.directory /app

CMD /app/scripts/ci.sh
