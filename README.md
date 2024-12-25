[![CI/CD Pipeline](https://github.com/waqar-ahmed48/DemoServer_ConnectionManager/actions/workflows/CICD%20Pipeline.yml/badge.svg)](https://github.com/waqar-ahmed48/DemoServer_ConnectionManager/actions/workflows/CICD%20Pipeline.yml)

## How to Build

- Just build: make
- Build & Run E2E Integration Tests: make config=test
    - It will build container image and run e2e integration tests with container image.
- Build & Run E2E Integration Tests with coverage: make config=testcoverage
    - It will run the application instance directly through go run and generate coverage reports
