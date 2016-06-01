FROM debian:jessie

RUN apt-get update && apt-get install --yes ca-certificates

COPY /.build/artifacts/route-controller /usr/bin/route-controller

CMD /usr/bin/route-controller

