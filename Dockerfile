FROM golang
WORKDIR /tmp/build
RUN git clone https://github.com/SUSE/eirinix-sample && \
    cd eirinix-sample && \
    go build

ENTRYPOINT ["/tmp/build/eirinix-sample/eirinix-sample"]
