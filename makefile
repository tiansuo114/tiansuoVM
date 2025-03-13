USER=lilin

.PHONY: build
build:			## Build generate binary
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 go build -v -o dist/console tiansuoVM/cmd/console

.PHONY: build_debug
build_debug:			## Run generate binary
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 go build -v -gcflags="all=-N -l" -o dist/console tiansuoVM/cmd/console

.PHONY: build_image
build_image:
	@echo "+ $@"

.PHONY: run
run:			## Run generate binary
	@echo "+ $@"
	./dist/console   --rdb-host="172.23.194.127"   --rdb-port=3306   --rdb-user="root"   --rdb-password="root"   --rdb-dbname="async_vm"   --ldap-host="172.23.194.127"   --ldap-port=389   --ldap-base-dn="dc=example,dc=com"   --ldap-user-name="cn=admin,dc=example,dc=com"   --ldap-password="123456"   --kube-config-path="/root/.kube/config"   --vm-namespace="tiansuo-vm" --deleted-vm-retention-period=1

.PHONY: run_debug
run_debug:
	@echo "+ $@"
	dlv --listen=:2345 --headless=true --api-version=2 --check-go-version=false --accept-multiclient exec ./dist/console -- --rdb-host="172.23.194.127"   --rdb-port=3306   --rdb-user="root"   --rdb-password="root"   --rdb-dbname="async_vm"   --ldap-host="172.23.194.127"   --ldap-port=389   --ldap-base-dn="dc=example,dc=com"   --ldap-user-name="cn=admin,dc=example,dc=com"   --ldap-password="123456"   --kube-config-path="/root/.kube/config"   --vm-namespace="tiansuo-vm"   --deleted-vm-retention-period=1