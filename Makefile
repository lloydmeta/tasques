DOCKER_TAGS ?= latest
VCS_REF=`git rev-parse --short HEAD`
BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

build-docker-image:
	@tag_string=''; \
	 for tag in ${DOCKER_TAGS}; do tag_string="$$tag_string -t lloydmeta/tasques:$$tag"; done; \
	 docker build \
	 	--build-arg VCS_REF=${VCS_REF} \
	 	--build-arg BUILD_DATE=${BUILD_DATE} \
		 $$tag_string . -f docker/Dockerfile

push-docker-image:
	@for tag in ${DOCKER_TAGS}; do docker push lloydmeta/tasques:$$tag; done;

docker-push-webhooks:
	curl -X POST https://hooks.microbadger.com/images/lloydmeta/tasques/4Asi8kXF8QxKZbCb5cSdBWoWBbE=