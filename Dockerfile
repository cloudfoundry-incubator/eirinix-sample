FROM golang
WORKDIR /tmp/build
RUN git clone https://github.com/mudler/eirinix-sample-extension && \
    cd eirinix-sample-extension && \
    go build

ENTRYPOINT ["/tmp/build/eirinix-sample-extension/eirinix-sample-extension"]
