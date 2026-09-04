set shell := ["bash", "-cu"]

default: generate-all

serve:
    npx http-server ./dist/ -p 3000

clean:
	rm -rf ./sources ./dist

clone-all:
	go run ./cmd/wgsl-docs-build clone

generate-all: clone-all
	go run ./cmd/wgsl-docs-build generate

build-bevy: generate-all

deploy-prod:
	test -d ./dist || (echo "./dist does not exist; run 'just generate-all' first" >&2; exit 1)
	vercel build --prod --local-config vercel.deploy.json
	vercel deploy --prebuilt --prod --yes --archive=tgz

deploy-dev:
    vercel build
    vercel --prebuilt
