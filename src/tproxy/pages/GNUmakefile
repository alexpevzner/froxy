include ../../../Rules.mak

all_local:
	hugo
	go-bindata -o pages.gen.go -pkg pages -prefix public `find public -type d`
	rm -rf public

clean_local:
	rm -f tproxy.pages.tar.gz

