include ../../../../../Rules.mak

deps := $(shell find archetypes content layouts static -type f)
deps += config.toml

all_local: pages.gen.go

pages.gen.go: $(deps)
	hugo
	go-bindata -o pages.gen.go -pkg pages -prefix public `find public -type d`
	rm -rf public
	rm -rf resources

clean_local:
	rm -f tproxy.pages.tar.gz
	rm -rf pages.gen.go
