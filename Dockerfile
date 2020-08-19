FROM scratch

LABEL maintainer="https://github.com/bakito/batch-job-controller"
ENV TZ="Europe/Zurich" \
  LANG="en_US.UTF-8"
EXPOSE 8080
WORKDIR /opt/go/
USER 1001
CMD ["/opt/go/batch-job-controller"]

ADD batch-job-controller /opt/go/
