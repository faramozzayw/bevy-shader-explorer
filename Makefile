.DEFAULT_GOAL := build-bevy
.PHONY: clone-bevy

BRANCHES_VERSIONS = \
	release-0.15.3:bevy-0.15.3 \
	release-0.16.0:bevy-0.16.0

serve:
	npx http-server ./dist/ -p 3000

clean:
	rm -rf ./bevy-* ./dist

clone-bevy:
	@for entry in $(BRANCHES_VERSIONS); do \
		branch=$${entry%%:*}; \
		dir=$${entry##*:}; \
		if [ ! -d "$$dir" ]; then \
			echo "Cloning $$branch into $$dir..."; \
			git clone --branch $$branch --depth=1 https://github.com/bevyengine/bevy.git $$dir && \
			rm -rf $$dir/.git; \
		else \
			echo "$$dir already exists. Skipping..."; \
		fi \
	done

build-bevy: clone-bevy
	@for entry in $(BRANCHES_VERSIONS); do \
		branch=$${entry%%:*}; \
		dir=$${entry##*:}; \
		version=$$(echo $$dir | sed 's/bevy-//'); \
		go run main --source ./$$dir/ --outputDir ./dist --version $$version --sourceGithubURL https://github.com/bevyengine/bevy/tree/$$branch/; \
		echo ""; \
	done

deploy-prod:
	vercel build --prod
	vercel --prebuilt --prod

deploy-dev:
	vercel build
	vercel --prebuilt
