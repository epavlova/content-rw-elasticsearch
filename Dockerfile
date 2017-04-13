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
  && BUILDINFO_PACKAGE="github.com/Financial-Times/service-status-go/buildinfo." \
  && VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && echo $LDFLAGS \
  && go get -t ./... \
  && go build  -ldflags="${LDFLAGS}" \
  && mv content-rw-elasticsearch /content-rw-elasticsearch \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/content-rw-elasticsearch" ]