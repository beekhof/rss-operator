compile:
	PASSES=simple hack/test

e2e-clean:
#	kubectl -n testing delete svc,pods,sts --all
	-ssh root@192.168.124.10 -- kubectl -n testing delete svc,pods,rss,sts --all

e2e: e2e-clean
	PASSES=e2e E2E_TEST_SELECTOR=TestCreateClusterOnly hack/test

build: 
	PASSES="prep simple build" E2E_TEST_SELECTOR=TestCreateClusterOnly hack/test

all: build e2e-clean e2e

generated:
	./hack/k8s/codegen/update-generated.sh 

t: target 

target:
	make -C galera all

init: target deps generated all

deps:
	glide install --strip-vendor 