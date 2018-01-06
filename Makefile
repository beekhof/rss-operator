E2E_TEST_SELECTOR=TestCreateCluster
NS=testing
OPERATOR_IMAGE=quay.io/beekhof/rss-operator:latest
export KUBECONFIG=$(HOME)/.kube/config
export GOPATH=$(HOME)/go
export GREP_OPTIONS=--color=never

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
install: deps check build
	cp _output/bin/rss-operator /usr/local/bin/rss-operator

all: build push e2e-clean e2e

clean: e2e-clean

e2e-clean:
#	kubectl -n testing delete svc,pods,sts --all
	-ssh root@192.168.124.10 -- kubectl -n testing delete crd,deploy,rs,rss,sts,svc,pods --all
	sleep 10


e2e: test-quick e2e-clean
	@echo "Running tests: $(E2E_TEST_SELECTOR)"
	PASSES=e2e TEST_NAMESPACE=$(NS) OPERATOR_IMAGE=$(OPERATOR_IMAGE) E2E_TEST_SELECTOR="$(E2E_TEST_SELECTOR)" hack/test 

generated:
	-rm -rf pkg/generated
	-find pkg -name zz_generated.deepcopy.go #delete
	./hack/k8s/codegen/update-generated.sh 

check:
	@echo "Checking gofmt..."
	for file in $(shell ./go-list.sh); do o=`gofmt -l -s -d $$file`; if [ "x$$o" != x ]; then echo "$$o"; exit 1; fi; done
	@echo "Checking unused..."
	$(GOPATH)/bin/unused $(PKGS)
	./hack/k8s/codegen/update-generated.sh --verify-only 
	$(GOPATH)/bin/gosimple $(PKGS)

target:
	make -C apps/galera all

init: target deps generated all

deps:
	go get honnef.co/go/tools/cmd/gosimple
	go get honnef.co/go/tools/cmd/unused
	#glide install --strip-vendor 

ns:
	-kubectl create ns $(NS)
	-kubectl -n $(NS) create clusterrolebinding $(NS)-everything --clusterrole=cluster-admin --serviceaccount=$(NS):default

test: ns
	-kubectl -n $(NS) create -f apps/galera/deployment.yaml
	@echo "Waiting for the operator to become active"
	while [ "x$$(kubectl -n testing get crd | grep replicatedstatefulsets.clusterlabs.org)" = x ]; do sleep 5; /bin/echo -n .; done
	kubectl -n $(NS) logs -f $(shell kubectl -n $(NS) get po | grep rss-operator | awk '{print $$1}')

.PHONY: test