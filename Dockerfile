FROM alpine:latest

RUN apk --no-cache add ca-certificates git bash zsh

COPY dirvana /usr/bin/dirvana

RUN chmod +x /usr/bin/dirvana

ENTRYPOINT ["/usr/bin/dirvana"]
CMD ["--help"]
