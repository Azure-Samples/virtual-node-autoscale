DOCKER_IMAGE ?= lachlanevenson/vk-affinity-admission-controller
GIT_BRANCH ?= `git rev-parse --abbrev-ref HEAD`

ifeq ($(GIT_BRANCH), master)
	DOCKER_TAG = canary
else
	DOCKER_TAG = $(GIT_BRANCH)
endif

ifeq ($(GIT_BRANCH),)
	DOCKER_TAG = ${CIRCLE_TAG}
endif

docker_build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker_push:
	# Push to DockerHub
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
