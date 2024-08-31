# parameters
REMOTE_HOST=root@10.0.0.249
TARGET_DIR=/opt/lightsailMon
IMAGE_TAG=ghcr.io/septrum101/lightsailmon:dev

# Targets
clean:
	docker image prune -f

build: clean
	docker build -f Dockerfile -t $(IMAGE_TAG) .

deploy:
	docker save $(IMAGE_TAG) | ssh $(REMOTE_HOST) "docker load"

update:
	ssh $(REMOTE_HOST) "docker-compose -f $(TARGET_DIR)/docker-compose.yml up -d"
	ssh $(REMOTE_HOST) "docker image prune -f"

all: build deploy update
