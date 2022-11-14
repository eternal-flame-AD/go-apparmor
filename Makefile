test: test.escape test.go
	true

test.go:
	$(MAKE) test.prep
	APPARMOR_TEST_SHIM=1 go test -v ./...
	$(MAKE) test.clean

test.escape: bin/test/escape
	bin/test/escape
	$(MAKE) test.prep
	@sudo apparmor_parser test-profile
	bin/test/escape
	$(MAKE) test.clean

bin/test/escape:
	go build -o bin/test/escape tests/escape/*.go

test.prep:
	mkdir -p bin/test
	@sudo apparmor_parser -R tests/clear.profile &>/dev/null || true

test.clean:
	rm test-profile &>/dev/null || true
	rm **/test-profile &>/dev/null || true

.PHONY: test test.escape test.clean
