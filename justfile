set shell := ["bash", "-cu"]

bevy_versions := "0.15.0 0.15.1 0.15.2 0.15.3 0.15.4 0.16.0 0.16.1 0.16.2 0.17.0 0.17.1 0.17.2 0.17.3 0.18.0 0.18.1 0.19.0 0.19.1"
hanabi_versions := "0.15.0 0.15.1 0.16.0 0.17.0 0.18.0 0.19.0"

default: generate-all

serve:
    npx http-server ./dist/ -p 3000

clean:
	rm -rf ./sources ./dist

clone-all:
	mkdir -p ./sources/bevy ./sources/hanabi
	for version in {{ bevy_versions }}; do \
		dir="./sources/bevy/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "release-$version" --depth=1 https://github.com/bevyengine/bevy.git "$dir" && rm -rf "$dir/.git"; fi; \
	done
	for version in {{ hanabi_versions }}; do \
		dir="./sources/hanabi/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "v$version" --depth=1 https://github.com/djeedai/bevy_hanabi.git "$dir" && rm -rf "$dir/.git"; fi; \
	done
	if [ ! -d ./sources/aqua ]; then git clone --depth=1 https://github.com/sayhisam1/bevy-aqua.git ./sources/aqua && rm -rf ./sources/aqua/.git; fi

generate-all: clone-all
	go run . generate --project ./sources/aqua --output ./dist
	for version in {{ bevy_versions }}; do \
		go run . generate --project "./sources/bevy/$version" --output ./dist --version "$version"; \
	done
	for version in {{ hanabi_versions }}; do \
		go run . generate --project "./sources/hanabi/$version" --output ./dist --version "$version"; \
	done

build-bevy: generate-all

deploy-prod:
	test -d ./dist || (echo "./dist does not exist; run 'just generate-all' first" >&2; exit 1)
	vercel build --prod --local-config vercel.deploy.json
	vercel deploy --prebuilt --prod --yes --archive=tgz

deploy-dev:
    vercel build
    vercel --prebuilt
