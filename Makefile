E2E_TEST_SELECTOR=TestCreateCluster
TEST_NAMESPACE=testing
OPERATOR_IMAGE=quay.io/beekhof/galera-experiment:mac
KUBECONFIG=$$HOME/.kube/config

PKGS=$(shell go list ./cmd/... ./pkg/... | grep -v -e generated -e apis/galera/v1alpha1)
TEST_PKGS=$(shell go list ./test/... | grep -v -e generated -e apis/galera/v1alpha1)

build: quick prep
	hack/build/operator/build

# function listPkgs() {
# 	go list ./cmd/... ./pkg/... ./test/... | grep -v generated
# }

# function listFiles() {
# 	# pipeline is much faster than for loop
# 	listPkgs | xargs -I {} find "${GOPATH}/src/{}" -name '*.go' | grep -v generated
# }

quick:
	gosimple $(PKGS)

test-quick:
	gosimple $(TEST_PKGS)

prep:
	OPERATOR_IMAGE=$(OPERATOR_IMAGE) PASSES="prep" hack/test	

push: build
	@echo "Uploading to $(OPERATOR_IMAGE)"
	OPERATOR_IMAGE=$(OPERATOR_IMAGE) hack/build/docker_push
	@echo "$(OPERATOR_IMAGE) updated"

install:  build
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

t: target 

target:
	make -C apps/galera all

init: target deps generated all

deps:
	glide install --strip-vendor 