TESTS := \
	test-client-success \
	test-client-fail-unexist \
	test-client-fail-already-created \
	test-conn-success \
	test-conn-fail-unexist \
	test-conn-fail-already-created \
	test-channel-success \
	test-channel-fail-unexist \
	test-channel-fail-already-created \

.PHONY: help
help:
	@echo "$(MAKE) <target>"
	@echo "targets are:"
	@for t in $(TESTS); do echo "  $${t}"; done

$(TESTS):
	./$@.sh

.PHONY: all
all:
	@for t in $(TESTS); do\
	  echo -n $$t...; \
	  ./$$t.sh >log.$$t 2>&1; \
	  if [ $$? -eq 0 ]; then \
	    echo "ok"; \
	  else \
	    echo "fail"; \
	    exit 1; \
	  fi \
	done

.PHONY: clean
clean:
	rm -rf log.* *~
