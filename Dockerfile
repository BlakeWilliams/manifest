FROM alpine:latest
RUN adduser -D -u 1001 manifestuser
RUN apk update && apk add --no-cache go git

USER manifestuser
WORKDIR /manifest
COPY ./ ./

RUN go install -buildvcs=false ./cmd/manifest
ENV PATH="/home/manifestuser/go/bin:${PATH}"

WORKDIR /app

RUN git config --global --add safe.directory /app
RUN echo "#!/bin/sh" > /home/manifestuser/start_cmd.sh
RUN echo "BRANCH=\${1:-\$(git rev-parse --abbrev-ref HEAD)}" >> /home/manifestuser/start_cmd.sh
RUN echo "HASH=\${2:-\$(git rev-parse HEAD)}" >> /home/manifestuser/start_cmd.sh
# RUN echo "git fetch origin \$BRANCH \$HASH" >> /home/manifestuser/start_cmd.sh
# RUN echo "git diff origin/\$BRANCH...HEAD | manifest inspect --sha \$HASH --formatter github --strict" >> /home/manifestuser/start_cmd.sh
# RUN chmod +x /home/manifestuser/start_cmd.sh
# RUN echo "#!/bin/sh" > /home/manifestuser/start_cmd.sh
# #RUN echo "git fetch origin \$BRANCH \$HASH" >> /home/manifestuser/start_cmd.sh
# RUN echo "BRANCH=\$(git rev-parse --abbrev-ref HEAD)" >> /home/manifestuser/start_cmd.sh
# RUN echo "HASH=\$(git rev-parse HEAD)" >> /home/manifestuser/start_cmd.sh
RUN echo "git diff origin/\$BRANCH...HEAD | manifest inspect --sha \$HASH --formatter github --strict" >> /home/manifestuser/start_cmd.sh
RUN chmod +x /home/manifestuser/start_cmd.sh
CMD /home/manifestuser/start_cmd.sh