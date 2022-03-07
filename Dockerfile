from golang

run mkdir /app

ADD . /app

WORKDIR /app

RUN go build .

EXPOSE 5555

CMD ["/app/sunnyservice"]