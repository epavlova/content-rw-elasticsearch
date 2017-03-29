FROM alpine:3.4

ADD *.go /content-rw-elasticsearch/

RUN apk add --update bash \
  && apk --update add git bzr go ca-certificates \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/content-rw-elasticsearch" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv content-rw-elasticsearch/* $GOPATH/src/${REPO_PATH} \
  && rm -r content-rw-elasticsearch \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t ./... \
  && go build \
  && mv content-rw-elasticsearch /content-rw-elasticsearch \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/concept-rw-elasticsearch" ]