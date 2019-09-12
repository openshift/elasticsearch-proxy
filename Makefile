test:
	@for pkg in $(TEST_PKGS) ; do \
		go test $(TEST_OPTIONS) $$pkg  ; \
	done
.PHONY: test
