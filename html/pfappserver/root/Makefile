FILES = dist/index.html dist/css/* dist/js/* dist/js/*

.PHONY: all vendor dev dist

all:
	@echo "Use 'make vendor' to update external libraries, 'make dev' to generate the static JS/CSS files and 'make dist' to commit the distribution version."

vendor:
	npm install --unsafe-perm

dev:
	npm run build-debug

dist:
	npm run build
	git add -f $(FILES)
	@if ! git diff --cached --quiet --exit-code; then \
		git add --update dist; \
		git commit -m "(web admin) Update generated files"; \
	else \
		echo "Nothing to commit; skipping git-commit"; \
	fi

light-dist:
	npm run build

clean:
	rm -rf ./node_modules
