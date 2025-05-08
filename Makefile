.PHONY: clone-bevy

serve:
	serve ./dist

clean:
	rm -rf ./bevy && rm -rf ./dist

clone-bevy:
	@if [ ! -d "bevy" ]; then \
		git clone --branch release-0.15.3 --depth=1 https://github.com/bevyengine/bevy.git && \
		rm -rf ./bevy/.git; \
	fi

build-bevy: clone-bevy
	go run main --source ./bevy/

deploy-prod:
	vercel build --prod
	vercel --prebuilt --prod
