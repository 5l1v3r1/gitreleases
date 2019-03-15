TAG?=$(shell git rev-parse --short=8 HEAD)
export TAG

all: build-landingpage build

install: install-go-deps install-npm
install-go-deps:
	go get -u github.com/rakyll/statik
install-npm:
	npm install --no-package-lock postcss-cli purgecss cssnano autoprefixer

# -- Go related targets
test:
	go test ./...
build:
	go generate
	go build -ldflags "-X main.version=$(TAG)" -o gitreleases .

# -- Docker build
pack:
	docker build -t registry.gitlab.com/mweibel/gitreleases:$(TAG) .
push:
	docker push registry.gitlab.com/mweibel/gitreleases
apply-secret:
	kubectl apply -f k8s/secret.yml
deploy:
	cat k8s/deployment.yml | sed 's/{{TAG}}/$(TAG)/g'| kubectl apply -f -
ship: test pack push deploy

# -- Landingpage build
build-landingpage: public/style.min.css public/index.html public/script.js | public

public:
	mkdir -p $@

public/style.min.css: style.css
	npx purgecss --css style.css --content landingpage/index.html --out .
	npx postcss style.css -o $@
	rm style.css
style.css: landingpage/gitreleases.css landingpage/tachyons.min.css
	cat landingpage/gitreleases.css landingpage/tachyons.min.css > $@

public/%.html: landingpage/%.html
	cp $< $@

public/%.js: landingpage/%.js
	cp $< $@


.PHONY: all install install-go-deps install-npm test build build-landingpage pack push apply-secret deploy ship