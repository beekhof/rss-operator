E2E_TEST_SELECTOR=TestCreateCluster
TEST_NAMESPACE=testing
OPERATOR_IMAGE=quay.io/beekhof/galera-experiment:mac
KUBECONFIG=$$HOME/.kube/config

PKGS=$(shell go list ./cmd/... ./pkg/... | grep -v -e generated -e apis/galera/v1alpha1)
TEST_PKGS=$(shell go list ./test/... | grep -v -e generated -e apis/galera/v1alpha1)

quick:
	gosimple $(PKGS)

build: 
	hack/build/operator/build

test-quick:
	gosimple $(TEST_PKGS)

push: check build
	@echo "building container..."
	docker build --tag "${OPERATOR_IMAGE}" -f hack/build/Dockerfile .
	@echo "Uploading to $(OPERATOR_IMAGE)"
	docker push $(OPERATOR_IMAGE)
	@echo "upload complete"

# Called from Dockerfile
install: deps build
	cp _output/bin/rss-operator /usr/local/bin/rss-operator

all: build push e2e-clean e2e

clean: e2e-clean

e2e-clean:
#	kubectl -n testing delete svc,pods,sts --all
	-ssh root@192.168.124.10 -- kubectl -n testing delete crd,rs,deploy,rss,sts,svc,pods --all
	sleep 10

e2e: test-quick e2e-clean
	@echo "Running tests: $(E2E_TEST_SELECTOR)"
	PASSES=e2e KUBECONFIG=$(KUBECONFIG) TEST_NAMESPACE=$(TEST_NAMESPACE) OPERATOR_IMAGE=$(OPERATOR_IMAGE) E2E_TEST_SELECTOR="$(E2E_TEST_SELECTOR)" hack/test 

generated:
	-rm -rf pkg/generated
	-find pkg -name zz_generated.deepcopy.go #delete
	./hack/k8s/codegen/update-generated.sh 

check:
	@echo "Checking gofmt..."
	for file in $(shell ./go-list.sh); do o=`gofmt -l -s -d $$file`; if [ "x$$o" != x ]; then echo "$$o"; exit 1; fi; done
	@echo "Checking unused..."
	unused $(PKGS)
	./hack/k8s/codegen/update-generated.sh --verify-only 
	gosimple $(PKGS)

target:
	make -C apps/galera all

init: target deps generated all

deps:
	go get honnef.co/go/tools/cmd/gosimple
	go get honnef.co/go/tools/cmd/unused
	#glide install --strip-vendor 