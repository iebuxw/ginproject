FROM elasticsearch:7.17.15

ARG IK_VERSION=7.17.15

USER elasticsearch
RUN set -eux; \
    curl -L -o /tmp/ik.zip \
      https://get.infini.cloud/elasticsearch/analysis-ik/${IK_VERSION}; \
    /usr/share/elasticsearch/bin/elasticsearch-plugin install --batch file:///tmp/ik.zip; \
    rm -f /tmp/ik.zip
