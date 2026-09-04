set shell := ["bash", "-cu"]

bevy_versions := "0.15.0 0.15.1 0.15.2 0.15.3 0.15.4 0.16.0 0.16.1 0.16.2 0.17.0 0.17.1 0.17.2 0.17.3 0.18.0 0.18.1 0.19.0 0.19.1"
hanabi_versions := "0.15.0 0.15.1 0.16.0 0.17.0 0.18.0 0.19.0"
# Entries are compatibility version -> upstream ref. Keep the compatibility
# version in the output even when a crate uses a different package version.
water_versions := "0.15.0:bevy_0.15 0.16.0:bevy_0.16 0.17.0:bevy_0.17 0.18.0:bevy_0.18"
outline_versions := "0.15.0:v0.9.2 0.16.0:bevy-0.16 0.17.0:bevy-0.17 0.18.0:bevy-0.18 0.19.0:bevy-0.19"
gaussian_versions := "0.15.0:v3.0.0 0.16.0:v5.1.0 0.17.0:v6.0.0"
atmosphere_versions := "0.11.0 0.12.0 0.13.0"

default: generate-all

serve:
    npx http-server ./dist/ -p 3000

clean:
	rm -rf ./sources ./dist

clone-all:
	mkdir -p ./sources/bevy ./sources/hanabi ./sources/bevy_water ./sources/bevy_mod_outline ./sources/bevy_gaussian_splatting ./sources/bevy_atmosphere
	for version in {{ bevy_versions }}; do \
		dir="./sources/bevy/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "release-$version" --depth=1 https://github.com/bevyengine/bevy.git "$dir" && rm -rf "$dir/.git"; fi; \
	done
	for version in {{ hanabi_versions }}; do \
		dir="./sources/hanabi/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "v$version" --depth=1 https://github.com/djeedai/bevy_hanabi.git "$dir" && rm -rf "$dir/.git"; fi; \
	done
	if [ ! -d ./sources/aqua ]; then git clone --depth=1 https://github.com/sayhisam1/bevy-aqua.git ./sources/aqua && rm -rf ./sources/aqua/.git; fi
	for spec in {{ water_versions }}; do \
		version="${spec%%:*}"; ref="${spec#*:}"; dir="./sources/bevy_water/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "$ref" --depth=1 https://github.com/Neopallium/bevy_water.git "$dir" && rm -rf "$dir/.git"; fi; \
	done
	for spec in {{ outline_versions }}; do \
		version="${spec%%:*}"; ref="${spec#*:}"; dir="./sources/bevy_mod_outline/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "$ref" --depth=1 https://github.com/komadori/bevy_mod_outline.git "$dir" && rm -rf "$dir/.git"; fi; \
	done
	for spec in {{ gaussian_versions }}; do \
		version="${spec%%:*}"; ref="${spec#*:}"; dir="./sources/bevy_gaussian_splatting/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "$ref" --depth=1 https://github.com/mosure/bevy_gaussian_splatting.git "$dir" && rm -rf "$dir/.git"; fi; \
	done
	for version in {{ atmosphere_versions }}; do \
		dir="./sources/bevy_atmosphere/$version"; \
		if [ ! -d "$dir" ]; then git clone --branch "$version" --depth=1 https://github.com/JonahPlusPlus/bevy_atmosphere.git "$dir" && rm -rf "$dir/.git"; fi; \
	done

generate-all: clone-all
	go run . generate --project ./sources/aqua --output ./dist
	for version in {{ bevy_versions }}; do \
		go run . generate --project "./sources/bevy/$version" --output ./dist --version "$version"; \
	done
	for version in {{ hanabi_versions }}; do \
		go run . generate --project "./sources/hanabi/$version" --output ./dist --version "$version"; \
	done
	for spec in {{ water_versions }}; do \
		version="${spec%%:*}"; \
		go run . generate --project "./sources/bevy_water/$version" --output ./dist --version "$version"; \
	done
	for spec in {{ outline_versions }}; do \
		version="${spec%%:*}"; \
		go run . generate --project "./sources/bevy_mod_outline/$version" --output ./dist --version "$version"; \
	done
	for spec in {{ gaussian_versions }}; do \
		version="${spec%%:*}"; \
		go run . generate --project "./sources/bevy_gaussian_splatting/$version" --output ./dist --version "$version"; \
	done
	for version in {{ atmosphere_versions }}; do \
		go run . generate --project "./sources/bevy_atmosphere/$version" --output ./dist --version "$version"; \
	done

build-bevy: generate-all

deploy-prod:
	test -d ./dist || (echo "./dist does not exist; run 'just generate-all' first" >&2; exit 1)
	vercel build --prod --local-config vercel.deploy.json
	vercel deploy --prebuilt --prod --yes --archive=tgz

deploy-dev:
    vercel build
    vercel --prebuilt
