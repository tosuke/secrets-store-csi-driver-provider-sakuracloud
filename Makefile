.PHONY: e2e-test
e2e-test:
	cd e2e && bats sakuracloud.bats
