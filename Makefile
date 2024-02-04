# Define variables
IMAGE_NAME_CONSUMER := lhconsumer
IMAGE_NAME_PRODUCER := lhproducer
TAG := latest

# Default target
all: docker_build_consumer docker_build_producer

# Target to build the Docker image for lhconsumer
docker_build_consumer:
	docker build -t $(IMAGE_NAME_CONSUMER):$(TAG) .

# Target to remove the built Docker image for lhconsumer
docker_clean_consumer:
	docker rmi $(IMAGE_NAME_CONSUMER):$(TAG)

# Target to build the Docker image for lhproducer
docker_build_producer:
	docker build -f Dockerfile.lhproducer -t $(IMAGE_NAME_PRODUCER):$(TAG) .

# Target to remove the built Docker image for lhproducer
docker_clean_producer:
	docker rmi $(IMAGE_NAME_PRODUCER):$(TAG)

.PHONY: all docker_build_consumer docker_clean_consumer docker_build_producer docker_clean_producer