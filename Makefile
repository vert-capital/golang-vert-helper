TRIVY_IMAGE ?= aquasec/trivy:0.69.3

security-scan:
	docker run --rm \
	-v $(PWD):/project \
	-v trivy-cache:/root/.cache/trivy \
	$(TRIVY_IMAGE) fs \
	--scanners vuln,secret \
	--exit-code 0 \
	--include-dev-deps \
	--severity HIGH,CRITICAL,MEDIUM \
	--skip-files .env \
	/project/