.PHONY: e2e-test
e2e-test:
	cd e2e && TEST_ID=$$(date +%s) bats sakuracloud.bats
