.PHONY: build-docker-storage-test-image test-filesystems

DATABASUS_STORAGE_TEST_IMAGE ?= databasus-storage-test:local

build-docker-storage-test-image:
	docker build --tag $(DATABASUS_STORAGE_TEST_IMAGE) .

test-filesystems:
	@if [ -z "$(DATABASUS_IMAGE)" ]; then \
		$(MAKE) build-docker-storage-test-image; \
	fi
	@DATABASUS_IMAGE="$${DATABASUS_IMAGE:-$(DATABASUS_STORAGE_TEST_IMAGE)}" \
		CASE="$(CASE)" \
		bash e2e/docker-storage/run.sh
