#MINIKUBE_IP := $(shell minikube ip)

ifeq ($(config), testdocker)
.DEFAULT_GOAL := testdocker
else ifeq ($(config), testk8s)
.DEFAULT_GOAL := testk8s
else ifeq ($(config), testcoverage)
.DEFAULT_GOAL := testcoverage
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
	#@echo  "Linting..."
	#golangci-lint run ./... --config=./lint/.golangci.yml
	
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
	#docker-compose down || true
	#docker-compose -f ./postgres/docker-compose.yml down || true

testk8s: runk8s
	@echo "---------------------------------------------------------------------------"
	@echo "-------------------------------- Test in K8s ------------------------------"
	@echo "---------------------------------------------------------------------------"
	#kill stack if already running
	helm delete demoserver_connectionmanager --wait || true
	helm delete my-postgres-release --wait || true
	kubectl delete secret demoserver-connectionmanager-postgres || true

	#bring up stack
	kubectl create secret generic demoserver-connectionmanager-postgres --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_HOST=my-postgres-release-postgresql-ha-pgpool.default.svc.cluster.local --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_PORT=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_PORT} --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_USERNAME=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_USERNAME} --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_USERNAME=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_USERNAME} --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_PASSWORD=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_PASSWORD} --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD} --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_CONNECTIONPOOLSIZE=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_CONNECTIONPOOLSIZE} --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_CONNECTIONPOOLSIZE=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_CONNECTIONPOOLSIZE} --from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_SSLMODE=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_SSLMODE}
	helm install my-postgres-release oci://registry-1.docker.io/bitnamicharts/postgresql-ha --set global.postgresql.password=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD} --set global.postgresql.repmgrPassword=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD} --set global.pgpool.adminPassword=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD}  --set postgresql.maxConnections=1000 --wait
	helm install demoserver_connectionmanager demoserver_connectionmanager_helm_chart/ --wait

	until curl http://${DEMOSERVER_CONNECTIONMANAGER_SERVICE_IP}:${DEMOSERVER_CONNECTIONMANAGER_SERVICE_PORT}/v1/connectionmgmt/status; do printf '.';sleep 1;done

	#test all postive test cases
	#go test -mod=mod -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./... -json > TestResults-Positive.json || true
	go clean -testcache
	go test -mod=mod -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./...

	#bring DB down so before initiating PostgresDown Negative test cases
	helm delete my-postgres-release --wait
	#go test -mod=mod -timeout 300s -run TestEndtoEndSuite/TestNegative_PostgresDown_ ./... -json > TestResults-Negative_PostgresDown.json
	go clean -testcache
	go test -mod=mod -timeout 300s -run TestEndtoEndSuite/TestNegative_PostgresDown_ ./...

	helm delete demoserver_connectionmanager --wait
	kubectl delete secret demoserver-connectionmanager-postgres

endif

ifeq ($(config), testcoverage)
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

testcoverage: runcoverage
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
	go test -timeout 300s -skip TestEndtoEndSuite/TestNegative_PostgresDown_ ./...

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