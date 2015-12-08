FROM busybox:ubuntu-14.04

VOLUME /root/.meow

ENV MEOW_VERSION 1.3.4
ADD build/linux/MEOW /MEOW
RUN chmod u+x MEOW
