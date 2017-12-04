IMAGE="quay.io/beekhof/galera-experiment:dev"
all: build push
build:
	echo "building container..."
	docker build --tag "$(IMAGE)" -f Dockerfile . 

# For gcr users, do "gcloud docker -a" to have access.

push:
	echo "pushing container..."
	docker push "$(IMAGE)" 

export:
	docker save $(IMAGE)  | gzip > $(IMAGE).tar.gz 
