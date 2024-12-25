ifeq ($(config), testdocker)
.DEFAULT_GOAL := testdocker
else ifeq ($(config), testk8s)
.DEFAULT_GOAL := testk8s
else ifeq ($(config), teststandalone)
.DEFAULT_GOAL := teststandalone
else
.DEFAULT_GOAL := build
endif
 
check_install:
	#@echo  "go mod..."
	#go mod vendor
	#go mod tidy

	which swagger || go get -u github.com/go-swagger/go-swagger/cmd/swagger

swagger: check_install
	#cleanup logs from previous run
	rm -f ./__debug_*

    echo DEMOSERVER_CONNECTIONMANAGER_SERVICE_IP: ${DEMOSERVER_CONNECTIONMANAGER_SERVICE_IP}
	echo DEMOSERVER_CONNECTIONMANAGER_SERVICE_PORT: ${DEMOSERVER_CONNECTIONMANAGER_SERVICE_PORT}
	#generate swagger docs yaml file
	#~/go/bin/swagger generate spec -o ./swagger.yaml --scan-models
	swagger generate spec -o ./swagger.yaml --scan-models

build: swagger
#	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
#	golangci-lint version

#	@echo  "Linting..."
#	golangci-lint cache clean
#	golangci-lint run ./... --config=./lint/.golangci.yml
#	golangci-lint run --skip-dirs='(e2e_test)' --config=./lint/.golangci.yml
	
	@echo  "Go build app..."
	go build -mod=mod
	chmod +x DemoServer_ConnectionManager

ifeq ($(config), testdocker)
rundocker: build
	#eval "$(docker-machine env -u)"
	@echo  "un-setting minikube docker env, in case needed."
	eval $(minikube docker-env -u)

	@echo  "building docker image"
	docker-compose build

	@echo  "kill any instance if already running"
	docker-compose down || true
	docker-compose -f ./postgres/docker-compose.yml down || true

testdocker: rundocker
	@echo "---------------------------------------------------------------------------"
	@echo "------------------------------- Test in Docker ----------------------------"
	@echo "---------------------------------------------------------------------------"

	@echo "bring up postgres"
	docker-compose -f ./postgres/docker-compose.yml up -d

	@echo "bring up application..."
	docker-compose up -d

	until curl http://${DEMOSERVER_CONNECTIONMANAGER_SERVICE_IP}:${DEMOSERVER_CONNECTIONMANAGER_SERVICE_PORT}/v1/connectionmgmt/status; do printf '.';sleep 1;done

	@echo "test all postive test cases"
	go clean -testcache
	go test -mod=mod -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./...

	@echo "bring postgres down so before initiating PostgresDown Negative test cases"
	docker-compose -f ./postgres/docker-compose.yml down
	go clean -testcache
	go test -mod=mod -timeout 300s -run TestEndtoEndSuite/TestNegative_PostgresDown_ ./...

	@echo "bring down stack"
	docker-compose down
	docker-compose -f ./postgres/docker-compose.yml down || true
endif

ifeq ($(config), testk8s)
runk8s: build
	#@eval $$(minikube docker-env)
	eval $(minikube docker-env)

	docker-compose build

	#cleanup logs from previous run
	rm -f ./e2e_test/coverage_reports/TestResults*

	#kill any instance if already running.
	docker-compose down || true
	docker-compose -f ./postgres/docker-compose.yml down || true

testk8s: runk8s

	@echo "---------------------------------------------------------------------------"
	@echo "-------------------------------- Test in K8s ------------------------------"
	@echo "---------------------------------------------------------------------------"
#	kill stack if already running
#	kubectl exec vault-0 --namespace vault-ns -- vault operator seal || true
	helm delete demoserver-connectionmanager --namespace demoserver --wait || true
	helm delete my-postgres-release --namespace demoserver --wait || true
	helm delete vault --namespace vault-ns --wait || true
	kubectl delete namespace demoserver --wait || true
	kubectl delete namespace vault-ns --wait || true
	kubectl delete -n demoserver secret demoserver-connectionmanager || true

#	bring up stack
	helm repo add hashicorp https://helm.releases.hashicorp.com || true
	kubectl create namespace vault-ns || true
	kubectl create namespace demoserver || true
	helm install vault hashicorp/vault -n vault-ns --wait || true

	@echo "calling vaultsetup script"
	./setup_vault_k8s.sh

	helm install -n demoserver my-postgres-release oci://registry-1.docker.io/bitnamicharts/postgresql-ha \
		--set global.postgresql.password=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD} \
		--set global.postgresql.repmgrPassword=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD} \
		--set global.pgpool.adminPassword=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD} \
		--set postgresql.maxConnections=1000 --wait

	helm install -n demoserver demoserver-connectionmanager demoserver_connectionmanager_helm_chart/ --wait

	until curl http://${DEMOSERVER_CONNECTIONMANAGER_SERVICE_IP}:${DEMOSERVER_CONNECTIONMANAGER_SERVICE_PORT}/v1/connectionmgmt/status; do printf '.';sleep 1;done

	#test all postive test cases
	#go test -mod=mod -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./... -json > TestResults-Positive.json || true
	go clean -testcache
	go test -mod=mod -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./...  -coverprofile=cover.out

	#bring DB down so before initiating PostgresDown Negative test cases
	helm delete my-postgres-release -n demoserver --wait
	#go test -mod=mod -timeout 300s -run TestEndtoEndSuite/TestNegative_PostgresDown_ ./... -json > TestResults-Negative_PostgresDown.json
	go clean -testcache
	go test -mod=mod -timeout 300s -run TestEndtoEndSuite/TestNegative_PostgresDown_ ./...

	helm delete demoserver-connectionmanager -n demoserver --wait
	kubectl delete secret demoserver-connectionmanager -n demoserver

endif

ifeq ($(config), teststandalone)
runcoverage: build
	#build intrumented
	go build -cover
	
	#cleanup logs from previous run
	rm -f ./e2e_test/coverage_reports/cov*
	rm -f ./e2e_test/coverage_reports/profile.txt
	rm -f ./e2e_test/coverage_reports/TestResults*

	#kill any instance if already running.
	pkill DemoServer_ConnectionManager || true
	docker-compose down || true

teststandalone: runcoverage
	@echo "---------------------------------------------------------------------------"
	@echo "------------------------------- Test Covrage ------------------------------"
	@echo "---------------------------------------------------------------------------"

	#bring up postgres
	docker-compose -f ./postgres/docker-compose.yml up -d

	#bring up application
	GOCOVERDIR=./e2e_test/coverage_reports ./DemoServer_ConnectionManager >DemoServer_ConnectionManager.log &
	
	until curl http://localhost:5678/v1/connectionmgmt/status; do printf '.';sleep 1;done

	#test all postive test cases
	#go test -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./... -json > ./e2e_test/coverage_reports/TestResults-Positive.json || true
	go clean -testcache
	go test -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./... -coverprofile=cover.out

	#bring postgres down so before initiating PostgresDown Negative test cases
	docker-compose -f ./postgres/docker-compose.yml down
	#go test -timeout 300s -run TestEndtoEndSuite/TestNegative_PostgresDown_ ./... -json > ./e2e_test/coverage_reports/TestResults-Negative_PostgresDown_.json || true
	go clean -testcache
	go test -timeout 300s -run TestEndtoEndSuite/TestNegative_PostgresDown_ ./...

	#bring down stack
	pkill -f -SIGINT DemoServer_ConnectionManager

	#generate coverage reports
	go tool covdata percent -i=./e2e_test/coverage_reports
	go tool covdata textfmt -i=./e2e_test/coverage_reports -o ./e2e_test/coverage_reports/profile.txt

endif